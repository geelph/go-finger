package output

import (
	"encoding/csv"
	"fmt"
	finger2 "gxx/pkg/finger"
	"gxx/types"
	"gxx/utils/proto"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	outputFile    *os.File
	csvWriter     *csv.Writer
	mu            sync.Mutex
	headerWritten bool
	// 预定义状态码映射，避免重复判断
	statusTextMap = map[string]string{
		"200": "OK",
		"201": "Created",
		"204": "No Content",
		"301": "Moved Permanently",
		"302": "Found",
		"304": "Not Modified",
		"400": "Bad Request",
		"401": "Unauthorized",
		"403": "Forbidden",
		"404": "Not Found",
		"500": "Internal Server Error",
		"502": "Bad Gateway",
		"503": "Service Unavailable",
	}
)

// WriteOptions 定义写入选项结构体，用于传递写入参数
type WriteOptions struct {
	Output      string            // 输出文件路径
	Format      string            // 输出格式(csv/txt)
	Target      string            // 目标URL
	Fingers     []*finger2.Finger // 指纹列表
	StatusCode  int32             // 状态码
	Title       string            // 页面标题
	ServerInfo  *types.ServerInfo // 服务器信息
	RespHeaders map[string]string // 响应头
	Response    *proto.Response   // 完整响应对象(可选)
	FinalResult bool              // 最终匹配结果
	Remark      string            // 备注(可选)
}

// InitOutput 初始化输出文件，写入表头
func InitOutput(outputPath, format string) error {
	if outputPath == "" {
		return nil
	}
	return openOutputFile(outputPath, format)
}

// WriteHeader 写入输出文件的表头
func WriteHeader(format string) error {
	if headerWritten || outputFile == nil {
		return nil
	}

	if format == "csv" {
		if csvWriter == nil {
			csvWriter = csv.NewWriter(outputFile)
		}

		// 写入扩展的CSV表头
		if err := csvWriter.Write([]string{
			"URL", "状态码", "标题", "服务器信息", "指纹ID", "指纹名称", "响应头", "匹配结果", "备注",
		}); err != nil {
			return fmt.Errorf("写入CSV表头失败: %v", err)
		}
		csvWriter.Flush()
	} else {
		// 文本格式表头
		header := fmt.Sprintf("%-40s%-10s%-30s%-20s%-30s%-30s%-50s%-15s%-20s\n",
			"URL", "状态码", "标题", "服务器信息", "指纹ID", "指纹名称", "响应头", "匹配结果", "备注")

		// 写入表头和分隔线
		if _, err := outputFile.WriteString(header); err != nil {
			return fmt.Errorf("写入表头失败: %v", err)
		}

		if _, err := outputFile.WriteString(strings.Repeat("-", 245) + "\n"); err != nil {
			return fmt.Errorf("写入分隔线失败: %v", err)
		}
	}

	headerWritten = true
	return nil
}

// openOutputFile 打开或创建输出文件的通用函数
func openOutputFile(output, format string) error {
	// 如果文件已经正确打开，直接返回
	if outputFile != nil && outputFile.Name() == output {
		return nil
	}

	// 关闭现有的文件
	if outputFile != nil {
		if csvWriter != nil {
			csvWriter.Flush()
		}
		_ = outputFile.Close()
		outputFile = nil
		csvWriter = nil
	}

	// 确保输出目录存在
	dir := filepath.Dir(output)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %v", err)
		}
	}

	// 检查文件是否存在
	fileExists := false
	if _, err := os.Stat(output); err == nil {
		fileExists = true
	}

	// 打开文件（追加模式）
	file, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开输出文件失败: %v", err)
	}

	outputFile = file
	headerWritten = fileExists

	// 初始化CSV写入器
	if format == "csv" {
		csvWriter = csv.NewWriter(file)
	}

	// 如果是新文件，写入表头
	if !fileExists {
		if err := WriteHeader(format); err != nil {
			return err
		}
	}

	return nil
}

// WriteFingerprints 使用结构体选项写入指纹组合结果
func WriteFingerprints(opts *WriteOptions) error {
	// 检查参数有效性
	if opts.Output == "" || len(opts.Fingers) == 0 {
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	// 确保文件已打开
	if err := openOutputFile(opts.Output, opts.Format); err != nil {
		return err
	}

	// 收集指纹信息并格式化
	fingersCount := len(opts.Fingers)
	fingerIDs := make([]string, 0, fingersCount)
	fingerNames := make([]string, 0, fingersCount)

	for _, f := range opts.Fingers {
		fingerIDs = append(fingerIDs, f.Id)
		fingerNames = append(fingerNames, f.Info.Name)
	}

	fingerIDStr := fmt.Sprintf("[%s]", strings.Join(fingerIDs, "，"))
	fingerNameStr := fmt.Sprintf("[%s]", strings.Join(fingerNames, "，"))

	// 使用传入的备注或生成默认备注
	remark := opts.Remark
	if remark == "" {
		remark = fmt.Sprintf("发现%d个指纹", fingersCount)
	}

	// 处理服务器信息
	serverInfoStr := ""
	if opts.ServerInfo != nil {
		serverInfoStr = opts.ServerInfo.ServerType
	}
	// 获取并合并响应头信息 - 预分配合理的容量
	headersCapacity := 10
	if opts.Response != nil && opts.Response.Headers != nil {
		headersCapacity = len(opts.Response.Headers) + 2 // 额外为Status和Protocol预留空间
	} else if opts.RespHeaders != nil {
		headersCapacity = len(opts.RespHeaders) + 2
	}

	headers := make(map[string]string, headersCapacity)

	// 如果提供了完整响应对象，优先使用其中的headers
	if opts.Response != nil && opts.Response.Headers != nil {
		for k, v := range opts.Response.Headers {
			headers[k] = v
		}

		// 添加状态码到头信息
		headers["Status"] = fmt.Sprintf("%d", opts.Response.Status)

		// 尝试从response中提取协议信息
		if len(opts.Response.Raw) > 0 {
			rawStr := string(opts.Response.Raw)
			if strings.HasPrefix(rawStr, "HTTP/2") {
				headers["Protocol"] = "HTTP/2"
			} else if strings.HasPrefix(rawStr, "HTTP/1.1") {
				headers["Protocol"] = "HTTP/1.1"
			}
		}
	}

	// 合并传入的自定义响应头
	if opts.RespHeaders != nil {
		for k, v := range opts.RespHeaders {
			headers[k] = v
		}
	}

	// 若状态码未设置，使用传入的状态码
	if _, exists := headers["Status"]; !exists && opts.StatusCode > 0 {
		headers["Status"] = fmt.Sprintf("%d", opts.StatusCode)
	}

	// 若Server头未设置，从ServerInfo添加
	if _, exists := headers["Server"]; !exists && opts.ServerInfo != nil && opts.ServerInfo.OriginalServer != "" {
		headers["Server"] = opts.ServerInfo.OriginalServer
	}

	// 格式化响应头为HTTP标准格式
	headersStr := formatHeaders(headers)

	// 写入结果
	if opts.Format == "csv" {
		if err := csvWriter.Write([]string{
			opts.Target,
			fmt.Sprintf("%d", opts.StatusCode),
			opts.Title,
			serverInfoStr,
			fingerIDStr,
			fingerNameStr,
			strings.ReplaceAll(headersStr, "\n", "\\n"), // CSV中换行符需要转义
			fmt.Sprintf("%v", opts.FinalResult),
			remark,
		}); err != nil {
			return fmt.Errorf("写入CSV记录失败: %v", err)
		}
		csvWriter.Flush()
	} else {
		// 使用strings.Builder提高字符串拼接效率
		var sb strings.Builder
		// 预分配合理的缓冲区大小
		sb.Grow(512 + len(headersStr))

		sb.WriteString("URL: ")
		sb.WriteString(opts.Target)
		sb.WriteString("\n状态码: ")
		sb.WriteString(fmt.Sprintf("%d", opts.StatusCode))
		sb.WriteString("\n标题: ")
		sb.WriteString(opts.Title)
		sb.WriteString("\n服务器: ")
		sb.WriteString(serverInfoStr)
		sb.WriteString("\n指纹ID: ")
		sb.WriteString(fingerIDStr)
		sb.WriteString("\n指纹名称: ")
		sb.WriteString(fingerNameStr)
		sb.WriteString("\n匹配结果: ")
		sb.WriteString(fmt.Sprintf("%v", opts.FinalResult))
		sb.WriteString("\n备注: ")
		sb.WriteString(remark)
		sb.WriteString("\n响应头:\n")
		sb.WriteString(headersStr)
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("-", 100))
		sb.WriteString("\n")

		if _, err := outputFile.WriteString(sb.String()); err != nil {
			return fmt.Errorf("写入结果失败: %v", err)
		}
	}

	return nil
}

// formatHeaders 将响应头格式化为标准HTTP头格式
func formatHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}

	// 预分配足够的容量
	var sb strings.Builder
	sb.Grow(256) // 预估合理的初始容量

	// 首先添加状态行
	statusCode := "200"
	if status, exists := headers["Status"]; exists {
		statusCode = status
		delete(headers, "Status") // 从headers中移除，避免重复显示
	}

	// 构建HTTP协议和状态行
	protocol := "HTTP/1.1"
	if proto, exists := headers["Protocol"]; exists {
		protocol = proto
		delete(headers, "Protocol")
	}

	// 获取状态码文本，使用预定义map替代多个if判断
	statusText, exists := statusTextMap[statusCode]
	if !exists {
		statusText = "OK" // 默认值
	}

	// 添加状态行
	fmt.Fprintf(&sb, "%s %s %s\n", protocol, statusCode, statusText)

	// 添加常见重要响应头（按照常见顺序排序）
	orderedHeaders := []string{
		"Date", "Server", "Content-Type", "Content-Length",
		"Last-Modified", "ETag", "Cache-Control", "Expires",
		"X-Powered-By", "Set-Cookie",
	}

	// 首先添加重要的响应头
	for _, key := range orderedHeaders {
		if value, exists := headers[key]; exists && value != "" {
			fmt.Fprintf(&sb, "%s: %s\n", key, value)
			delete(headers, key) // 从map中删除已处理的头
		}
	}

	// 然后添加剩余的响应头（按字母顺序）
	// 预分配剩余键的容量
	remainingKeys := make([]string, 0, len(headers))
	for key := range headers {
		// 直接在循环中过滤非标准HTTP头
		if key != "Title" && key != "Version" && key != "ServerType" {
			remainingKeys = append(remainingKeys, key)
		}
	}
	sort.Strings(remainingKeys)

	for _, key := range remainingKeys {
		fmt.Fprintf(&sb, "%s: %s\n", key, headers[key])
	}

	return sb.String()
}

// WriteResult 写入单个指纹结果到文件(兼容旧版接口)
func WriteResult(output, format, target string, fg *finger2.Finger, statusCode int32, title string, serverInfo *types.ServerInfo, respHeaders map[string]string, finalResult bool) error {
	// 转换为结构体选项
	opts := &WriteOptions{
		Output:      output,
		Format:      format,
		Target:      target,
		Fingers:     []*finger2.Finger{fg},
		StatusCode:  statusCode,
		Title:       title,
		ServerInfo:  serverInfo,
		RespHeaders: respHeaders,
		FinalResult: finalResult,
	}

	return WriteFingerprints(opts)
}

// Close 关闭输出文件
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if outputFile != nil {
		if csvWriter != nil {
			csvWriter.Flush()
		}
		err := outputFile.Close()
		outputFile = nil
		csvWriter = nil
		headerWritten = false
		return err
	}
	return nil
}
