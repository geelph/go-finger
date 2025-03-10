/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: yaml.go
    @Date: 2025/2/21 下午2:36*
*/
package finger

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"gxx/utils/common"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FingerFile 配置poc文件目录
const FingerFile = "finger"
const (
	HttpType = "http"
	TcpType  = "tcp"
	UdpType  = "udp"
	SslType  = "ssl"
	GoType   = "go"
)

var order = 0

type Finger struct {
	Id         string        `yaml:"id"`        //  脚本名称
	Transport  string        `yaml:"transport"` // 传输方式，该字段用于指定发送数据包的协议，该字段用于指定发送数据包的协议:①tcp ②udp ③http
	Set        yaml.MapSlice `yaml:"set"`       // 全局变量定义，该字段用于定义全局变量。比如随机数，反连平台等
	Payloads   Payloads      `yaml:"payloads"`
	Rules      RuleMapSlice  `yaml:"rules"`
	Expression string        `yaml:"expression"`
	Info       Info          `yaml:"info"`
	Gopoc      string        `yaml:"gopoc"` // Gopoc 脚本名称
}
type Payloads struct {
	Continue bool          `yaml:"continue"`
	Payloads yaml.MapSlice `yaml:"payloads"`
}

// RuleMap 用于帮助yaml解析，保证Rule有序
type RuleMap struct {
	Key   string
	Value Rule
}

// RuleMapSlice 用于帮助yaml解析，保证Rule有序
type RuleMapSlice []RuleMap

type Rule struct {
	Request        RuleRequest   `yaml:"request"`
	Expression     string        `yaml:"expression"`
	Expressions    []string      `yaml:"expressions"`
	Output         yaml.MapSlice `yaml:"output"`
	StopIfMatch    bool          `yaml:"stop_if_match"`
	StopIfMismatch bool          `yaml:"stop_if_mismatch"`
	BeforeSleep    int           `yaml:"before_sleep"`
	order          int
}
type RuleRequest struct {
	Type            string            `yaml:"type"`         // 传输方式，默认 http，可选：tcp,udp,ssl,go 等任意扩展
	Host            string            `yaml:"host"`         // tcp/udp 请求的主机名
	Data            string            `yaml:"data"`         // tcp/udp 发送的内容
	DataType        string            `yaml:"data-type"`    // tcp/udp 发送的数据类型，默认字符串
	ReadSize        int               `yaml:"read-size"`    // tcp/udp 读取内容的长度
	ReadTimeout     int               `yaml:"read-timeout"` // tcp/udp专用
	Raw             string            `yaml:"raw"`          // raw 专用
	Method          string            `yaml:"method"`
	Path            string            `yaml:"path"`
	Headers         map[string]string `yaml:"headers"`
	Body            string            `yaml:"body"`
	FollowRedirects bool              `yaml:"follow_redirects"` // 是否跟随重定向，默认跟随重定向
}

// Info 以下开始是 信息部分
type Info struct {
	Name           string         `yaml:"name"`
	Author         string         `yaml:"author"`
	Severity       string         `yaml:"severity"`
	Verified       bool           `yaml:"verified"`
	Description    string         `yaml:"description"`
	Reference      []string       `yaml:"reference"`
	Affected       string         `yaml:"affected"`  // 影响版本
	Solutions      string         `yaml:"solutions"` // 解决方案
	Tags           string         `yaml:"tags"`      // 标签
	Classification Classification `yaml:"classification"`
	Created        string         `yaml:"created"` // create time
}

type Classification struct {
	CvssMetrics string  `yaml:"cvss-metrics"`
	CvssScore   float64 `yaml:"cvss-score"`
	CveId       string  `yaml:"cve-id"`
	CweId       string  `yaml:"cwe-id"`
}

type ruleAlias struct {
	Request        RuleRequest   `yaml:"request"`
	Expression     string        `yaml:"expression"`
	Expressions    []string      `yaml:"expressions"`
	Output         yaml.MapSlice `yaml:"output"`
	StopIfMatch    bool          `yaml:"stop_if_match"`
	StopIfMismatch bool          `yaml:"stop_if_mismatch"`
	BeforeSleep    int           `yaml:"before_sleep"`
}

// GetFingerPath 获取poc文件目录
func GetFingerPath() string {
	homeDir, err := os.Getwd()
	if err != nil {
		return "1"
	}
	configFile := filepath.Join(homeDir, FingerFile)
	if !common.Exists(configFile) {
		return ""
	}
	return configFile
}

// Select 获取指定名字的yaml文件位置
func Select(pocPath string, pocName string) (string, error) {
	var result string
	// 遍历目录中的所有文件和子目录
	err := filepath.WalkDir(pocPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // 如果遇到错误，立即返回
		}

		// 检查文件是否是 YAML 文件
		if !d.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			if strings.Contains(d.Name(), pocName) {
				result = path
				return filepath.SkipDir // 找到文件后停止遍历
			}
		}
		return nil
	})

	if err != nil {
		return "", err // 如果遍历过程中出错，返回错误
	}

	if len(result) == 0 {
		return result, fmt.Errorf("未找到匹配的结果")
	}

	return result, err
}

// Read 获取yaml文件内容
func Read(FingerYaml string) (Finger, error) {
	var poc = Finger{}

	file, err := os.Open(FingerYaml)
	if err != nil {
		return poc, err
	}
	defer file.Close()

	if err := yaml.NewDecoder(file).Decode(&poc); err != nil {
		return poc, err
	}
	return poc, nil
}

// GetAll 获取特定目录下面所有yaml文件
func GetAll(root string) ([]string, error) {
	var allPocs []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			allPocs = append(allPocs, path)
		}
		return nil
	})

	return allPocs, err
}

// IsHTTPType 判断是否是http请求
func (finger *Finger) IsHTTPType() bool {
	for _, rule := range finger.Rules {
		reqType := rule.Value.Request.Type
		if len(reqType) == 0 || reqType == HttpType {
			return true
		}
	}
	return false
}

// UnmarshalYAML 解析yaml文件内容
func (r *Rule) UnmarshalYAML(unmarshal func(any) error) error {

	var tmp ruleAlias
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	r.Request = tmp.Request
	r.Expression = tmp.Expression
	r.Expressions = append(r.Expressions, tmp.Expressions...)
	r.Output = tmp.Output
	r.StopIfMatch = tmp.StopIfMatch
	r.StopIfMismatch = tmp.StopIfMismatch
	r.BeforeSleep = tmp.BeforeSleep
	r.order = order

	order += 1
	return nil
}

// UnmarshalYAML 解析yaml文件内容
func (m *RuleMapSlice) UnmarshalYAML(unmarshal func(any) error) error {
	order = 0

	tempMap := make(map[string]Rule, 1)
	err := unmarshal(&tempMap)
	if err != nil {
		return err
	}

	newRuleSlice := make([]RuleMap, len(tempMap))
	for roleName, role := range tempMap {
		newRuleSlice[role.order] = RuleMap{
			Key:   roleName,
			Value: role,
		}
	}

	*m = newRuleSlice
	return nil
}
