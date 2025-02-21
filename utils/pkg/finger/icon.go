/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: icon.go
    @Date: 2025/2/21 下午3:06*
*/
package finger

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/spaolacci/murmur3"
	"github.com/valyala/fasthttp"
	_ "github.com/vmihailenco/msgpack/v5"
	"gxx/utils/common"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// GetIconHash 用于获取icon hash的相关操作
type GetIconHash struct {
	iconURL    string
	retries    int
	headers    map[string]string
	fileHeader []string
}

// NewGetIconHash 初始化 GetIconHash
func NewGetIconHash(iconURL string, retries ...int) *GetIconHash {
	// 设置默认值为 1
	retriesValue := 1
	if len(retries) > 0 {
		retriesValue = retries[0]
	}
	return &GetIconHash{
		iconURL: iconURL,
		retries: retriesValue,
		headers: map[string]string{
			"Accept":        "application/x-shockwave-flash, image/gif, image/x-xbitmap, image/jpeg, image/pjpeg, application/vnd.ms-excel, application/vnd.ms-powerpoint, application/msword, */*",
			"User-agent":    common.RandomUA(),
			"Pragma":        "no-cache",
			"Cache-Control": "no-cache",
			"Connection":    "close",
		},
		fileHeader: []string{
			"89504E470", "89504e470", "000001000", "474946383", "FFD8FFE00", "FFD8FFE10", "3c7376672", "3c3f786d6",
		},
	}
}

// getDefaultIconURL 获取默认icon的URL
func (g *GetIconHash) getDefaultIconURL(iconURL string) string {
	parsedURL, err := url.Parse(iconURL)
	if err != nil {
		fmt.Println("URL解析错误: ", err)
		return ""
	}
	return fmt.Sprintf("%s://%s/favicon.ico", parsedURL.Scheme, parsedURL.Host)
}

// getIconHash 获取icon的hash值
func (g *GetIconHash) getIconHash(iconURL string) int32 {
	fmt.Println("icon url地址: ", iconURL)
	var iconHash int32

	if strings.HasPrefix(iconURL, "data:image/vnd.microsoft.icon") {
		iconHash = g.hashDataURL(iconURL)
	} else if strings.HasPrefix(iconURL, "http") {
		iconHash = g.hashHTTPURL(iconURL)
	}

	return iconHash
}

// hashDataURL 处理 data URL 并计算 hash 值
func (g *GetIconHash) hashDataURL(iconURL string) int32 {
	iconBase64List := strings.Split(iconURL, ";base64,")
	if len(iconBase64List) > 1 {
		iconBase64 := iconBase64List[1]
		iconData, err := base64.StdEncoding.DecodeString(iconBase64)
		if err != nil {
			fmt.Println("icon base64解密出错，错误信息: ", err)
			return 0
		}
		return Mmh3Hash32(iconData)
	}
	return 0
}

// hashHTTPURL 处理 HTTP URL 并计算 hash 值
func (g *GetIconHash) hashHTTPURL(iconURL string) int32 {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(iconURL)
	for key, value := range g.headers {
		req.Header.Set(key, value)
	}

	client := &fasthttp.Client{}
	if err := client.DoTimeout(req, resp, 30*time.Second); err != nil {
		fmt.Println("icon获取报错，错误信息: ", err)
		return 0
	}

	if resp.StatusCode() == fasthttp.StatusOK && len(resp.Body()) != 0 {
		for _, fh := range g.fileHeader {
			if strings.HasPrefix(fmt.Sprintf("%x", resp.Body()), fh) {
				return Mmh3Hash32(resp.Body())
			}
		}
	}
	return 0
}

// StandBase64 计算 base64 的值
func StandBase64(raw []byte) []byte {
	B64 := base64.StdEncoding.EncodeToString(raw)
	var buffer bytes.Buffer
	for i := 0; i < len(B64); i++ {
		buffer.WriteByte(B64[i])
		if (i+1)%76 == 0 {
			buffer.WriteByte('\n')
		}
	}
	buffer.WriteByte('\n')
	return buffer.Bytes()
}

// Mmh3Hash32 计算数据的哈希值
func Mmh3Hash32(raw []byte) int32 {
	base64Data := StandBase64(raw)
	var h32 = murmur3.New32()
	_, err := h32.Write(base64Data)
	if err != nil {
		return 0
	}
	fmt.Println("icon hash：", int32(h32.Sum32()))
	return int32(h32.Sum32())
}

// Run 主入口函数
func (g *GetIconHash) Run() string {
	iconHash := g.getIconHash(g.iconURL)
	if iconHash == 0 {
		defaultIconURL := g.getDefaultIconURL(g.iconURL)
		if defaultIconURL != "" {
			fmt.Println("重置为默认icon地址尝试请求: ", defaultIconURL)
			iconHash = g.getIconHash(defaultIconURL)
		}
	}
	return fmt.Sprintf("%d", iconHash)
}

// GetIconURL 获取icon的url地址
func GetIconURL(pageURL string, html string) string {
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		fmt.Println("URL解析错误: ", err)
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
		if len(match) > 1 {
			ext := strings.ToLower(strings.Split(match[1], ".")[1])
			if ext == "ico" || ext == "png" || ext == "jpg" || ext == "jpeg" || ext == "gif" || ext == "svg" || ext == "icon" {
				ic = append(ic, match[1])
			}
		}
	}

	if iconIndex == -1 && shortcutIndex == -1 {
		if len(ic) > 0 {
			faviconURL = baseURL + ic[0]
			fmt.Println("发现新icon地址", faviconURL)
			if basePath != "" {
				faviconPath := basePath + ic[0]
				faviconURL = urlJoin(baseURL, strings.TrimLeft(faviconPath, "/"))
				fmt.Println("有原始path已重置url", faviconURL)
			}
		} else if basePath != "" {
			faviconPath := basePath + "/favicon.ico"
			faviconURL = urlJoin(baseURL, strings.TrimLeft(faviconPath, "/"))
			fmt.Println("使用默认url+path", faviconURL)
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
			faviconURL = urlJoin(baseURL, strings.TrimLeft(faviconPath, "/"))
			fmt.Println("页面提取到icon url", faviconURL)
			if !strings.HasPrefix(faviconPath, basePath) && !strings.Contains(html, basePath[1:]+"/") {
				if strings.Contains(basePath, "log") || len(strings.Split(basePath, "/")) > 3 {
					basePathParts := strings.Split(strings.TrimLeft(basePath, "/"), "/")
					basePathParts = basePathParts[:len(basePathParts)-1]
					basePath = strings.Join(basePathParts, "/")
					fmt.Println("原始path过长or识别到login结尾自动修正，新path", basePath)
					faviconPath = basePath + "/" + strings.TrimLeft(faviconPath, "/")
				} else {
					faviconPath = basePath + "/" + strings.TrimLeft(faviconPath, "/")
				}
				faviconURL = urlJoin(baseURL, strings.TrimLeft(faviconPath, "/"))
				fmt.Println("有原始path且原始path不在favicon_path中已重置url", faviconURL)
			}
		}
	}

	return faviconURL
}

// urlJoin 拼接 baseURL 和相对路径
func urlJoin(baseURL string, relativePath string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		log.Println("URL解析错误: ", err)
		return ""
	}
	ref, err := url.Parse(relativePath)
	if err != nil {
		log.Println("URL解析错误: ", err)
		return ""
	}
	return base.ResolveReference(ref).String()
}
