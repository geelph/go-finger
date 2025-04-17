package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	finger2 "gxx/pkg/finger"
	"gxx/pkg/wappalyzer"
	"gxx/types"
	"gxx/utils/logger"
	"gxx/utils/proto"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	outputFile      *os.File
	csvWriter       *csv.Writer
	sockFile        *os.File       // socket文件句柄
	mu              sync.Mutex
	headerWritten   bool
	sockListener    net.Listener
	sockConnections = make(map[net.Conn]bool)
	sockConnMutex   sync.Mutex
)

// WriteOptions 定义写入选项结构体，用于传递写入参数
type WriteOptions struct {
	Output      string                     // 输出文件路径
	Format      string                     // 输出格式(csv/txt/json)
	Target      string                     // 目标URL
	Fingers     []*finger2.Finger          // 指纹列表
	StatusCode  int32                      // 状态码
	Title       string                     // 页面标题
	ServerInfo  *types.ServerInfo          // 服务器信息
	RespHeaders string                     // 响应头
	Response    *proto.Response            // 完整响应对象(可选)
	Wappalyzer  *wappalyzer.TypeWappalyzer //站点使用技术
	FinalResult bool                       // 最终匹配结果
	Remark      string                     // 备注(可选)
}

// JSONOutput JSON格式输出结构体
type JSONOutput struct {
	URL         string                     `json:"url"`
	StatusCode  int32                      `json:"status_code"`
	Title       string                     `json:"title"`
	Server      string                     `json:"server"`
	FingerIDs   []string                   `json:"finger_ids,omitempty"`
	FingerNames []string                   `json:"finger_names,omitempty"`
	Headers     string                     `json:"headers,omitempty"`
	Wappalyzer  *wappalyzer.TypeWappalyzer `json:"wappalyzer,omitempty"`
	MatchResult bool                       `json:"match_result"`
	Remark      string                     `json:"remark,omitempty"`
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
	} else if format == "json" {
		// JSON格式不需要写表头
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
	if opts.Output == "" {
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

	// 格式化响应头为HTTP标准格式
	headersStr := ""
	if opts.Response != nil && opts.Response.RawHeader != nil {
		headersStr = string(opts.Response.RawHeader)
	} else if opts.RespHeaders != "" {
		headersStr = opts.RespHeaders
	}

	// 根据不同格式写入结果
	if opts.Format == "json" {
		// 构建JSON对象
		jsonOutput := &JSONOutput{
			URL:         opts.Target,
			StatusCode:  opts.StatusCode,
			Title:       opts.Title,
			Server:      serverInfoStr,
			FingerIDs:   fingerIDs,
			FingerNames: fingerNames,
			Headers:     headersStr,
			Wappalyzer:  opts.Wappalyzer,
			MatchResult: opts.FinalResult,
			Remark:      remark,
		}

		// 序列化为JSON
		jsonData, err := json.MarshalIndent(jsonOutput, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON序列化失败: %v", err)
		}

		// 写入JSON数据和换行符
		if _, err := outputFile.Write(jsonData); err != nil {
			return fmt.Errorf("写入JSON数据失败: %v", err)
		}
		if _, err := outputFile.Write([]byte("\n")); err != nil {
			return fmt.Errorf("写入换行符失败: %v", err)
		}

	} else if opts.Format == "csv" {
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

// PrintJSONOutput 将结果直接以JSON格式输出到标准输出
func PrintJSONOutput(opts *WriteOptions) error {
	// 收集指纹信息
	fingersCount := len(opts.Fingers)
	fingerIDs := make([]string, 0, fingersCount)
	fingerNames := make([]string, 0, fingersCount)

	for _, f := range opts.Fingers {
		fingerIDs = append(fingerIDs, f.Id)
		fingerNames = append(fingerNames, f.Info.Name)
	}

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

	// 格式化响应头
	headersStr := ""
	if opts.Response != nil && opts.Response.RawHeader != nil {
		headersStr = string(opts.Response.RawHeader)
	} else if opts.RespHeaders != "" {
		headersStr = opts.RespHeaders
	}

	// 构建JSON对象
	jsonOutput := &JSONOutput{
		URL:         opts.Target,
		StatusCode:  opts.StatusCode,
		Title:       opts.Title,
		Server:      serverInfoStr,
		FingerIDs:   fingerIDs,
		FingerNames: fingerNames,
		Headers:     headersStr,
		Wappalyzer:  opts.Wappalyzer,
		MatchResult: opts.FinalResult,
		Remark:      remark,
	}

	// 序列化为JSON
	jsonData, err := json.MarshalIndent(jsonOutput, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	// 输出到标准输出
	fmt.Println(string(jsonData))
	return nil
}

// WriteResult 写入单个指纹结果到文件(兼容旧版接口)
func WriteResult(output, format, target string, fg *finger2.Finger, statusCode int32, title string, serverInfo *types.ServerInfo, respHeaders string, finalResult bool) error {
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

	var err error

	// 关闭常规输出文件
	if outputFile != nil {
		if csvWriter != nil {
			csvWriter.Flush()
		}
		err = outputFile.Close()
		outputFile = nil
		csvWriter = nil
		headerWritten = false
	}

	// 关闭socket文件
	if sockFile != nil {
		sockFile = nil
	}
	
	// 关闭socket监听器
	if sockListener != nil {
		if closeErr := sockListener.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		
		// 关闭所有连接
		sockConnMutex.Lock()
		for conn := range sockConnections {
			_ = conn.Close()
		}
		sockConnections = make(map[net.Conn]bool)
		sockConnMutex.Unlock()
		
		sockListener = nil
	}

	return err
}

// InitSockOutput 初始化socket文件输出
func InitSockOutput(sockPath string) error {
	if sockPath == "" {
		return nil
	}

	// 如果已经有socket监听，先关闭
	if sockFile != nil {
		_ = sockFile.Close()
		sockFile = nil
	}

	// 确保输出目录存在
	dir := filepath.Dir(sockPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建socket输出目录失败: %v", err)
		}
	}

	// 删除已存在的socket文件（如果存在）
	_ = os.Remove(sockPath)

	// 创建Unix domain socket监听
	unixListener, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("创建Unix domain socket失败: %v", err)
	}

	// 启动协程接受连接并处理
	go func() {
		for {
			conn, err := unixListener.Accept()
			if err != nil {
				// 如果监听已关闭，退出循环
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				logger.Error(fmt.Sprintf("Unix socket接受连接失败: %v", err))
				continue
			}

			// 对每个连接启动一个协程处理
			go handleConnection(conn)
		}
	}()

	// 保存监听器，以便后续关闭
	sockFile = &os.File{} // 用于保持与接口兼容性
	sockListener = unixListener

	return nil
}

// handleConnection 处理单个socket连接
func handleConnection(conn net.Conn) {
	// 添加到连接集合
	sockConnMutex.Lock()
	sockConnections[conn] = true
	sockConnMutex.Unlock()

	// 函数返回时清理连接
	defer func() {
		sockConnMutex.Lock()
		delete(sockConnections, conn)
		_ = conn.Close()
		sockConnMutex.Unlock()
	}()

	// 保持连接打开
	buffer := make([]byte, 1024)
	for {
		_, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				logger.Debug(fmt.Sprintf("Unix socket读取错误: %v", err))
			}
			return
		}
	}
}

// WriteToSock 将结果以JSON格式写入所有socket连接
func WriteToSock(opts *WriteOptions) error {
	if sockListener == nil {
		return nil
	}

	// 收集指纹信息
	fingersCount := len(opts.Fingers)
	fingerIDs := make([]string, 0, fingersCount)
	fingerNames := make([]string, 0, fingersCount)

	for _, f := range opts.Fingers {
		fingerIDs = append(fingerIDs, f.Id)
		fingerNames = append(fingerNames, f.Info.Name)
	}

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

	// 格式化响应头
	headersStr := ""
	if opts.Response != nil && opts.Response.RawHeader != nil {
		headersStr = string(opts.Response.RawHeader)
	} else if opts.RespHeaders != "" {
		headersStr = opts.RespHeaders
	}

	// 构建JSON对象
	jsonOutput := &JSONOutput{
		URL:         opts.Target,
		StatusCode:  opts.StatusCode,
		Title:       opts.Title,
		Server:      serverInfoStr,
		FingerIDs:   fingerIDs,
		FingerNames: fingerNames,
		Headers:     headersStr,
		Wappalyzer:  opts.Wappalyzer,
		MatchResult: opts.FinalResult,
		Remark:      remark,
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(jsonOutput)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	// 添加换行符
	jsonData = append(jsonData, '\n')

	// 向所有连接写入数据
	sockConnMutex.Lock()
	for conn := range sockConnections {
		_, _ = conn.Write(jsonData)
	}
	sockConnMutex.Unlock()

	return nil
}
