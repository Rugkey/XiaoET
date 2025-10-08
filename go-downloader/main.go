package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var httpClient *http.Client
var headerCookie string
var headerReferer string
var numRetries int
var reqTimeout time.Duration

func initHTTP() {
	httpClient = &http.Client{Timeout: reqTimeout}
}

func doGetToFile(url, out string) error {
	var lastErr error
	for attempt := 0; attempt < numRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "goose-batch-downloader/1.0")
		if headerCookie != "" {
			req.Header.Set("Cookie", headerCookie)
		}
		if headerReferer != "" {
			req.Header.Set("Referer", headerReferer)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				f, err := os.Create(out)
				if err != nil {
					return err
				}
				_, err = io.Copy(f, resp.Body)
				cerr := f.Close()
				if err != nil {
					return err
				}
				if cerr != nil {
					return cerr
				}
				return nil
			}
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
		}
		time.Sleep(time.Duration(attempt+1) * 300 * time.Millisecond)
	}
	return lastErr
}

// 下载m3u8文件
func downloadM3U8(url string) (string, error) {
	baseName := filepath.Base(url)
	dir := strings.TrimSuffix(baseName, ".m3u8")
	err := os.Mkdir(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return "", err
	}
	err = os.Chdir(dir)
	if err != nil {
		return "", err
	}
	if err := doGetToFile(url, baseName); err != nil {
		return "", err
	}
	return baseName, nil
}

// 获取ts下载文件
func getUrls(prefix, filename string) ([]byte, []byte, []string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, nil, nil, err
	}

	// 解析 KEY 与可选 IV
	keyLineRE := regexp.MustCompile(`#EXT-X-KEY:METHOD=AES-128,URI="(.*?)"(?:,IV=(0x[0-9A-Fa-f]+))?`)
	keyLine := keyLineRE.FindSubmatch(data)
	if len(keyLine) < 2 {
		return nil, nil, nil, errors.New("failed to match key url")
	}
	keyURL := string(keyLine[1])
	var iv []byte
	if len(keyLine) >= 3 && len(keyLine[2]) > 0 {
		hexStr := strings.TrimPrefix(string(keyLine[2]), "0x")
		b, err := hex.DecodeString(hexStr)
		if err == nil {
			if len(b) < 16 {
				pad := make([]byte, 16-len(b))
				b = append(pad, b...)
			}
			if len(b) > 16 {
				b = b[len(b)-16:]
			}
			iv = b
		}
	}

	// 解析url
	tsUrlRE := regexp.MustCompile(`.+\.ts.+`)
	tsUrls := tsUrlRE.FindAllSubmatch(data, -1)
	if len(tsUrls) == 0 {
		return nil, nil, nil, errors.New("failed to match ts urls")
	}

	urls := make([]string, len(tsUrls))
	for i, lv1 := range tsUrls {
		u := string(lv1[0])
		if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
			urls[i] = u
		} else {
			urls[i] = prefix + u
		}
	}

	// 下载 key
	if err := doGetToFile(keyURL, "key"); err != nil {
		return nil, nil, nil, err
	}
	key, err := ioutil.ReadFile("./key")
	if err != nil {
		return nil, nil, nil, err
	}

	return key, iv, urls, nil
}

type task struct {
	num int
	url string
}

type result struct {
	num      int
	url      string
	filename string
	err      error
}

func downloadChunks(key []byte, iv []byte, urls []string) (int, error) {
	downloadCh := make(chan task, len(urls))
	resultCh := make(chan result, len(urls))
	var wg sync.WaitGroup
	wg.Add(len(urls))

	// 发出下载任务
	for i, url := range urls {
		downloadCh <- task{
			num: i,
			url: url,
		}
	}
	close(downloadCh)

	// 启动10个协程下载文件
	fmt.Println("total: ", len(urls))
	for i := 0; i < 10; i++ {
		go func() {
			for t := range downloadCh {
				// 下载文件
				fmt.Printf("downloading %d %s\n", t.num, t.url)
				filename := strconv.Itoa(t.num) + ".ts"
				err := doGetToFile(t.url, filename)
				if err != nil {
					fmt.Println("error", err)
				}
				resultCh <- result{
					num:      t.num,
					url:      t.url,
					filename: filename,
					err:      err,
				}
				wg.Done()
			}
		}()
	}

	wg.Wait()

	// 检查结果
	fmt.Println(len(resultCh))
	for i := 0; i < len(urls); i++ {
		rs := <-resultCh
		if rs.err != nil {
			return 0, fmt.Errorf("failed to download %s: %v", rs.url, rs.err)
		}

		// 解密
		if len(key) > 0 {
			block, err := aes.NewCipher(key)
			if err != nil {
				return 0, err
			}
			fileData, err := ioutil.ReadFile(rs.filename)
			if err != nil {
				return 0, err
			}
			pt := make([]byte, len(fileData))
			useIV := iv
			if len(useIV) != 16 {
				useIV = bytes.Repeat([]byte{0}, 16)
			}
			bm := cipher.NewCBCDecrypter(block, useIV)
			bm.CryptBlocks(pt, fileData)
			err = ioutil.WriteFile(rs.filename, pt, 0755)
			if err != nil {
				return 0, err
			}
		}
	}

	return len(urls), nil
}

func mergeFile(count int) error {
	// ffmpeg -i "concat:ttt.ts|tt2.ts" -c copy output.ts
	files := make([]string, count)
	for i := range files {
		files[i] = strconv.Itoa(i) + ".ts"
	}

	// Too many open files
	// ulimit -n 1024

	fmt.Println("ffmpeg", "-i", fmt.Sprintf("\"concat:%s\"", strings.Join(files, "|")), "-c", "copy", "merge.ts")

	cmd := exec.Command("ffmpeg", "-i", fmt.Sprintf("concat:%s", strings.Join(files, "|")), "-c", "copy", "merge.ts")
	o, e := cmd.CombinedOutput()
	fmt.Println(string(o))
	return e
}

func getPrefix(url string) string {
	i := strings.LastIndex(url, "/")
	return url[:i+1]
}

func main() {
	var url string
	var newName string
	var listFile string
	var prefix string
	var cookie string
	var referer string
	var retries int
	var timeoutSec int
	flag.StringVar(&url, "u", "", "m3u8 url")
	flag.StringVar(&newName, "n", "", "new name (single) or base name (batch)")
	flag.StringVar(&listFile, "list", "", "path to file containing multiple m3u8 urls (one per line)")
	flag.StringVar(&prefix, "prefix", "", "output name prefix for batch downloads")
	flag.StringVar(&cookie, "cookie", "", "Cookie header value")
	flag.StringVar(&referer, "referer", "", "Referer header value")
	flag.IntVar(&retries, "retries", 3, "max retries per request")
	flag.IntVar(&timeoutSec, "timeout", 30, "request timeout seconds")
	flag.Parse()

	// 初始化 HTTP 客户端与头
	headerCookie = cookie
	headerReferer = referer
	numRetries = retries
	reqTimeout = time.Duration(timeoutSec) * time.Second
	initHTTP()

	// 单个下载执行函数
	downloadOne := func(u string, name string) error {
		found, err := regexp.MatchString("m3u8($|\\?.*)", u)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("please enter valid m3u8 url: %s", u)
		}

		// 1. 下载m3u8
		filename, err := downloadM3U8(u)
		if err != nil {
			return err
		}

		// 2. 解析出key和分片url
		key, iv, tsUrls, err := getUrls(getPrefix(u), filename)
		if err != nil {
			return err
		}

		// 3. 并发下载分片文件并解密
		count, err := downloadChunks(key, iv, tsUrls)
		if err != nil {
			return err
		}

		// 4. 合并文件
		err = mergeFile(count)
		fmt.Println(err)
		if err != nil {
			return err
		}

		// 5. 移动输出并清理目录
		if len(name) > 0 {
			err = os.Rename("merge.ts", "../"+name+".ts")
		} else if len(newName) > 0 {
			// 回退到全局 -n
			err = os.Rename("merge.ts", "../"+newName+".ts")
		} else {
			err = os.Rename("merge.ts", "../merge.ts")
		}
		fmt.Println("move", err)
		if err == nil {
			os.Chdir("../")
			err = os.RemoveAll(strings.TrimSuffix(filepath.Base(u), ".m3u8"))
			fmt.Println("remove", err)
		}
		return err
	}

	// 批量模式
	if len(listFile) > 0 {
		data, err := ioutil.ReadFile(listFile)
		if err != nil {
			panic(err)
		}
		lines := strings.Split(string(data), "\n")
		idx := 0
		for _, line := range lines {
			u := strings.TrimSpace(line)
			if u == "" || strings.HasPrefix(u, "#") {
				continue
			}
			idx++
			// 生成输出名：优先 prefix，其次 -n，加序号
			var name string
			if len(prefix) > 0 {
				name = fmt.Sprintf("%s_%03d", prefix, idx)
			} else if len(newName) > 0 {
				name = fmt.Sprintf("%s_%03d", newName, idx)
			} else {
				name = ""
			}
			fmt.Printf("\n=== [%d] downloading %s ===\n", idx, u)
			if err := downloadOne(u, name); err != nil {
				fmt.Printf("error on %s: %v\n", u, err)
			}
		}
		return
	}

	// 单个模式
	if len(url) == 0 {
		fmt.Println("please provide -u url or -list file")
		return
	}
	if err := downloadOne(url, ""); err != nil {
		panic(err)
	}
}
