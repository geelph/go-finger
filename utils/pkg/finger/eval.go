/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: eval.go
    @Date: 2025/2/21 下午3:01*
*/
package finger

import (
	"fmt"
	"github.com/google/cel-go/checker/decls"
	"gopkg.in/yaml.v2"
	"gxx/utils/common"
	"gxx/utils/config"
	cels "gxx/utils/pkg/cel"
	"gxx/utils/pkg/proto"
	"net/url"
	"strings"
)

type Checker struct {
	VariableMap map[string]any
	CustomLib   *cels.CustomLib
}

type ListItem struct {
	Key   string
	Value []string
}
type ListMap []ListItem

// IsFuzzSet 解析Set中的定义变量
func IsFuzzSet(args yaml.MapSlice, variableMap map[string]any, customLib *cels.CustomLib) {
	for _, arg := range args {
		key := arg.Key.(string)
		value := arg.Value.(string)
		// 处理dns反连
		if value == "newReverse()" {
			variableMap[key] = newReverse()
			customLib.UpdateCompileOption(key, decls.NewObjectType("proto.Reverse"))
			continue
		}
		// 处理jndi连接
		if value == "newJNDI()" {
			variableMap[key] = newJNDI()
			customLib.UpdateCompileOption(key, decls.NewObjectType("proto.Reverse"))
			continue
		}

		out, err := customLib.Evaluate(value, variableMap)
		if err != nil {
			variableMap[key] = fmt.Sprintf("%v", value)
			customLib.UpdateCompileOption(key, decls.String)
			continue
		}
		switch value := out.Value().(type) {
		case *proto.UrlType:
			variableMap[key] = common.UrlTypeToString(value)
			customLib.UpdateCompileOption(key, decls.NewObjectType("proto.UrlType"))
		case int64:
			variableMap[key] = int(value)
			customLib.UpdateCompileOption(key, decls.Int)
		case map[string]string:
			variableMap[key] = value
			customLib.UpdateCompileOption(key, cels.StrStrMapType)
		default:
			variableMap[key] = fmt.Sprintf("%v", out)
			customLib.UpdateCompileOption(key, decls.String)
		}
	}
}

// SetVariableMap 处理解析set中变量
func SetVariableMap(find string, variableMap map[string]any) string {
	for k, v := range variableMap {
		_, isMap := v.(map[string]string)
		if isMap {
			continue
		}
		newStr := fmt.Sprintf("%v", v)
		oldStr := "{{" + k + "}}"
		if !strings.Contains(find, oldStr) {
			continue
		}
		find = strings.ReplaceAll(find, oldStr, newStr)
	}
	return find
}

// newReverse 处理dns反连
func newReverse() *proto.Reverse {
	sub := common.RandomString(12)
	urlStr := fmt.Sprintf("http://%s.%s", sub, config.ReverseCeyeDomain)
	u, _ := url.Parse(urlStr)
	return &proto.Reverse{
		Url:                common.ParseUrl(u),
		Domain:             u.Hostname(),
		Ip:                 u.Host,
		IsDomainNameServer: false,
	}
}

// newJNDI 处理jndi连接
func newJNDI() *proto.Reverse {
	randomStr := common.RandomString(22)
	urlStr := fmt.Sprintf("http://%s:%s/%s", config.ReverseJndi, config.ReverseLdapPort, randomStr)
	u, _ := url.Parse(urlStr)
	parseUrl := common.ParseUrl(u)
	return &proto.Reverse{
		Url:                parseUrl,
		Domain:             u.Hostname(),
		Ip:                 config.ReverseJndi,
		IsDomainNameServer: false,
	}
}
