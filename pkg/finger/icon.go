/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: icon.go
    @Date: 2025/2/21 下午3:06*
*/
package finger

import (
	"crypto/tls"
	"fmt"
	"gxx/utils/common"
	"gxx/utils/logger"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/chainreactors/proxyclient"
	"github.com/spaolacci/murmur3"
	_ "github.com/vmihailenco/msgpack/v5"
)

// GetIconHash 获取icon hash
type GetIconHash struct {
	iconURL    string
	retries    int
	headers    map[string]string
	fileHeader []string
	proxy      string
}

// NewGetIconHash 初始化 GetIconHash
func NewGetIconHash(iconURL string, proxy string, retries ...int) *GetIconHash {
	// 设置默认值为 1
	retriesValue := 1
	if len(retries) > 0 {
		retriesValue = retries[0]
	}

	return &GetIconHash{
		iconURL: iconURL,
		retries: retriesValue,
		headers: map[string]string{
			"User-Agent":      common.RandomUA(),
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.5",
			"Connection":      "close",
		},
		fileHeader: []string{
			"89504E470", "89504e470", "000001000", "474946383", "FFD8FFE00", "FFD8FFE10", "3c7376672", "3c3f786d6",
		},
		proxy: proxy,
	}
}

// getDefaultIconURL 获取默认的icon URL
func (g *GetIconHash) getDefaultIconURL(iconURL string) string {
	if iconURL == "" {
		return ""
	}
	parsedURL, err := url.Parse(iconURL)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s://%s/favicon.ico", parsedURL.Scheme, parsedURL.Host)
}

// getIconHash 获取icon的hash值
func (g *GetIconHash) getIconHash(iconURL string) int32 {
	if strings.HasPrefix(iconURL, "data:") {
		return g.hashDataURL(iconURL)
	}
	return g.hashHTTPURL(iconURL)
}

// hashDataURL 处理 data URL 并计算 hash 值
func (g *GetIconHash) hashDataURL(iconURL string) int32 {
	parts := strings.Split(iconURL, ",")
	if len(parts) != 2 {
		return 0
	}
	iconData := StandBase64([]byte(parts[1]))
	if len(iconData) != 0 {
		return Mmh3Hash32(iconData)
	}
	return 0
}

// hashHTTPURL 处理 HTTP URL 并计算 hash 值
func (g *GetIconHash) hashHTTPURL(iconURL string) int32 {
	iconURL = iconURL + "?time=" + fmt.Sprintf("%d%d", time.Now().Unix(), rand.New(rand.NewSource(time.Now().UnixNano())).Intn(10000))

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// 如果有代理，使用proxyclient处理代理请求
	if g.proxy != "" {
		parsedURL, err := url.Parse(g.proxy)
		if err != nil {
			logger.Error(fmt.Sprintf("代理地址解析失败：%s", err))
			return 0
		}
		proxyClient, err := proxyclient.NewClient(parsedURL)
		if err != nil {
			logger.Error(fmt.Sprintf("创建代理客户端失败：%s", err))
			return 0
		}

		client.Transport = &http.Transport{
			DialContext:     proxyClient.DialContext,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// 创建请求
	req, err := http.NewRequest("GET", iconURL, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("创建请求失败: %s", err))
		return 0
	}

	// 设置请求头
	for key, value := range g.headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		logger.Debug(fmt.Sprintf("icon获取报错，错误信息：%s", err))
		return 0
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// 读取响应体
	var bodyBytes []byte
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			logger.Debug(fmt.Sprintf("读取响应体失败: %s", err))
			return 0
		}

		// 检查文件头
		if len(bodyBytes) > 0 {
			bodyHex := fmt.Sprintf("%x", bodyBytes[:8])
			for _, fh := range g.fileHeader {
				if strings.HasPrefix(bodyHex, strings.ToLower(fh)) {
					return Mmh3Hash32(bodyBytes)
				}
			}
		}
	}

	return 0
}

// StandBase64 标准化Base64编码
func StandBase64(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte{}
	}
	data := make([]byte, len(raw))
	copy(data, raw)

	for i, b := range data {
		if b == '_' {
			data[i] = '/'
		} else if b == '-' {
			data[i] = '+'
		}
	}
	return data
}

// Mmh3Hash32 计算Mmh3Hash32哈希值
func Mmh3Hash32(raw []byte) int32 {
	hasher := murmur3.New32()
	_, _ = hasher.Write(raw)
	return int32(hasher.Sum32())
}

// Run 运行获取icon hash的流程
func (g *GetIconHash) Run() string {
	var hash int32
	if g.iconURL != "" {
		hash = g.getIconHash(g.iconURL)
	}
	if hash == 0 {
		defaultURL := g.getDefaultIconURL(g.iconURL)
		if defaultURL != "" {
			hash = g.getIconHash(defaultURL)
		}
	}
	return fmt.Sprintf("%d", hash)
}

// GetIconURL 获取icon的url地址
func GetIconURL(pageURL string, html string) string {
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		logger.Error(fmt.Sprintf("URL解析错误: %s", err))
		return ""
	}

	baseURL := fmt.Sprintf("%s://%s/", parsedURL.Scheme, parsedURL.Host)
	basePath := parsedURL.Path
	if strings.Contains(basePath, ".") || strings.Contains(basePath, ".htm") {
		basePath = ""
	}

	faviconURL := baseURL + "favicon.ico"
	htmlLower := strings.ToLower(html)
	iconIndex := strings.Index(htmlLower, "<link rel=\"icon\"")
	shortcutIndex := strings.Index(htmlLower, "<link rel=\"shortcut icon\"")

	re := regexp.MustCompile(`href="(.*?)"`)
	iconList := re.FindAllStringSubmatch(html, -1)
	var ic []string
	for _, match := range iconList {
		if len(match) > 1 && strings.Contains(match[1], ".") {
			parts := strings.Split(match[1], ".")
			if len(parts) > 1 {
				ext := strings.ToLower(parts[len(parts)-1])
				if ext == "ico" || ext == "png" || ext == "jpg" || ext == "jpeg" || ext == "gif" || ext == "svg" || ext == "icon" {
					ic = append(ic, match[1])
				}
			}
		}
	}

	if iconIndex == -1 && shortcutIndex == -1 {
		if len(ic) > 0 {
			// 检查是否是完整URL
			if strings.HasPrefix(ic[0], "http://") || strings.HasPrefix(ic[0], "https://") {
				faviconURL = ic[0]
			} else if strings.HasPrefix(ic[0], "//") {
				// 处理协议相对URL
				faviconURL = parsedURL.Scheme + ":" + ic[0]
			} else if strings.HasPrefix(ic[0], "/") {
				// 绝对路径
				faviconURL = baseURL + strings.TrimPrefix(ic[0], "/")
			} else {
				// 相对路径
				if basePath == "" || strings.HasSuffix(basePath, "/") {
					faviconURL = baseURL + strings.TrimPrefix(basePath, "/") + ic[0]
				} else {
					// 如果basePath不以/结尾，需要获取目录部分
					dir := path.Dir(basePath)
					if dir == "." {
						dir = ""
					} else if !strings.HasSuffix(dir, "/") {
						dir += "/"
					}
					faviconURL = baseURL + strings.TrimPrefix(dir, "/") + ic[0]
				}
			}
			logger.Debug(fmt.Sprintf("发现新icon地址：%s", faviconURL))
		} else if basePath != "" {
			faviconPath := basePath
			if !strings.HasSuffix(faviconPath, "/") {
				faviconPath = path.Dir(faviconPath)
				if faviconPath != "." && !strings.HasSuffix(faviconPath, "/") {
					faviconPath += "/"
				}
			}
			faviconURL = baseURL + strings.TrimPrefix(strings.TrimPrefix(faviconPath, "/"), "./") + "favicon.ico"
			logger.Debug(fmt.Sprintf("使用默认url+path：%s", faviconURL))
		}
	} else {
		var linkTag string
		if iconIndex != -1 {
			linkTag = html[iconIndex : strings.Index(html[iconIndex:], ">")+iconIndex]
		} else {
			linkTag = html[shortcutIndex : strings.Index(html[shortcutIndex:], ">")+shortcutIndex]
		}

		reHref := regexp.MustCompile(`href="([^"]+)"`)
		faviconPathMatch := reHref.FindStringSubmatch(linkTag)
		if len(faviconPathMatch) > 1 {
			faviconPath := faviconPathMatch[1]

			// 检查是否是完整URL
			if strings.HasPrefix(faviconPath, "http://") || strings.HasPrefix(faviconPath, "https://") {
				faviconURL = faviconPath
			} else if strings.HasPrefix(faviconPath, "//") {
				// 处理协议相对URL
				faviconURL = parsedURL.Scheme + ":" + faviconPath
			} else if strings.HasPrefix(faviconPath, "/") {
				// 绝对路径
				faviconURL = baseURL + strings.TrimPrefix(faviconPath, "/")
			} else {
				// 相对路径
				if basePath == "" || strings.HasSuffix(basePath, "/") {
					faviconURL = baseURL + strings.TrimPrefix(basePath, "/") + faviconPath
				} else {
					// 如果basePath不以/结尾，需要获取目录部分
					dir := path.Dir(basePath)
					if dir == "." {
						dir = ""
					} else if !strings.HasSuffix(dir, "/") {
						dir += "/"
					}
					faviconURL = baseURL + strings.TrimPrefix(dir, "/") + faviconPath
				}
			}
			logger.Debug(fmt.Sprintf("页面提取到icon url %s", faviconURL))
		}
	}

	return faviconURL
}
