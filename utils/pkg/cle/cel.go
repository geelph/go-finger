/*
  - Package cel
    @Author: zhizhuo
    @IDE：GoLand
    @File: cel.go
    @Date: 2025/2/7 上午8:57*
*/
package cel

import (
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"golang.org/x/net/context"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"gopkg.in/yaml.v2"
	"gxx/utils/logger"
	"strings"
	"time"
)

// CustomLib 自定义库结构体
type CustomLib struct {
	envOptions     []cel.EnvOption
	programOptions []cel.ProgramOption
}

func (c *CustomLib) CompileOptions() []cel.EnvOption {
	return c.envOptions
}

func (c *CustomLib) ProgramOptions() []cel.ProgramOption {
	return c.programOptions
}

// Evaluate 执行规则判断
func (c *CustomLib) Evaluate(expression string, variable map[string]any) (ref.Val, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	var (
		val ref.Val
		err error
	)
	resp := make(chan int)
	go func() {
		defer close(resp)

		env, err := c.NewCelEnv()
		if err != nil {
			resp <- 9
		}
		val, err = Eval(env, expression, variable)
		if err != nil {
			resp <- 9
		}
		resp <- 99
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("eval timed out")
	case v := <-resp:
		if v == 99 {
			return val, err
		}
		return nil, fmt.Errorf("eval error")
	}
}

// NewCustomLib 创建新的 CustomLib 实例
func NewCustomLib() *CustomLib {
	c := &CustomLib{}
	reg := types.NewEmptyRegistry()
	c.envOptions = ReadCompileOptions(reg)
	c.programOptions = ReadProgramOptions(reg)
	return c
}

// NewCelEnv 创建新的 CEL 环境
func (c *CustomLib) NewCelEnv() (env *cel.Env, err error) {
	env, err = cel.NewEnv(cel.Lib(c))
	return env, err
}

// Eval 执行 CEL 表达式
func Eval(env *cel.Env, expression string, params map[string]any) (ref.Val, error) {
	ast, iss := env.Compile(expression)
	if iss.Err() != nil {
		logger.Error(fmt.Sprintf("cel env.Compile err, %s", iss.Err().Error()))
		return nil, iss.Err()
	}
	prg, err := env.Program(ast)
	if err != nil {
		logger.Error(fmt.Sprintf("cel env.Program err, %s", err.Error()))
		return nil, err
	}
	out, _, err := prg.Eval(params)
	if err != nil {
		fmt.Println("cel error --> ", err.Error())
		logger.Error(fmt.Sprintf("cel prg.Eval err, %s", err.Error()))
		return nil, err
	}
	return out, nil
}

// WriteRuleSetOptions 写入规则集选项
func (c *CustomLib) WriteRuleSetOptions(args yaml.MapSlice) {
	for _, v := range args {
		key := v.Key.(string)
		value := v.Value

		var d *exprpb.Decl
		switch vv := value.(type) {
		case int64:
			d = decls.NewVar(key, decls.Int)
		case string:
			if strings.HasPrefix(vv, "newReverse") {
				d = decls.NewVar(key, decls.NewObjectType("proto.Reverse"))
			} else if strings.HasPrefix(vv, "randomInt") {
				d = decls.NewVar(key, decls.Int)
			} else {
				d = decls.NewVar(key, decls.String)
			}
		case map[string]string:
			d = decls.NewVar(key, StrStrMapType)
		default:
			d = decls.NewVar(key, decls.String)
		}
		c.envOptions = append(c.envOptions, cel.Declarations(d))
	}
}

// WriteRuleFunctionsROptions 自定函数用于处理r0 || r1规则解析
func (c *CustomLib) WriteRuleFunctionsROptions(funcName string, returnBool bool) {
	// 使用 cel.Function 方法注册函数及其重载
	c.envOptions = append(c.envOptions, cel.Function(
		funcName,
		cel.Overload(
			funcName+"_bool", // 重载名，通常为函数名加上类型后缀
			[]*cel.Type{},    // 参数类型列表，这里为空表示函数无参数
			cel.BoolType,     // 返回类型为布尔类型
			cel.FunctionBinding(func(values ...ref.Val) ref.Val {
				// 自定义函数的实现，这里简单地返回传入的布尔值
				return types.Bool(returnBool)
			}),
		),
	))
}

// UpdateCompileOption 更新编译选项
func (c *CustomLib) UpdateCompileOption(k string, t *exprpb.Type) {
	c.envOptions = append(c.envOptions, cel.Declarations(decls.NewVar(k, t)))
}

// Reset 重置 CustomLib 实例
func (c *CustomLib) Reset() {
	*c = CustomLib{}
}

// WriteRuleIsVulOptions 写入漏洞检测选项
func (c *CustomLib) WriteRuleIsVulOptions(key string, isVul bool) {
	c.envOptions = append(c.envOptions, cel.Declarations(decls.NewVar(key+"()", decls.Bool)))
}
