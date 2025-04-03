/*
  - Package cel
    @Author: zhizhuo
    @IDE：GoLand
    @File: cel.go
    @Date: 2025/2/7 上午8:57*
*/
package cel

import (
	"context"
	"fmt"
	"gxx/utils/logger"
	"strings"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"gopkg.in/yaml.v2"
)

// CustomLib 自定义CEL库结构体
type CustomLib struct {
	envOptions     []cel.EnvOption
	programOptions []cel.ProgramOption
	mu             sync.Mutex
}

// CompileOptions 返回环境选项
func (c *CustomLib) CompileOptions() []cel.EnvOption {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.envOptions
}

// ProgramOptions 返回程序选项
func (c *CustomLib) ProgramOptions() []cel.ProgramOption {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.programOptions
}

// Evaluate 执行CEL表达式并返回结果
func (c *CustomLib) Evaluate(expression string, variables map[string]any) (ref.Val, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	type result struct {
		val ref.Val
		err error
	}
	resultCh := make(chan result, 1)

	go func() {
		// 为每个goroutine创建新的环境
		env, err := cel.NewEnv(cel.Lib(c))
		if err != nil {
			resultCh <- result{nil, fmt.Errorf("创建CEL环境失败: %w", err)}
			return
		}

		val, err := Eval(env, expression, variables)
		resultCh <- result{val, err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("表达式执行超时")
	case res := <-resultCh:
		return res.val, res.err
	}
}

// NewCustomLib 创建新的CustomLib实例
func NewCustomLib() *CustomLib {
	c := &CustomLib{}
	reg := types.NewEmptyRegistry()
	c.envOptions = ReadCompileOptions(reg)
	c.programOptions = ReadProgramOptions(reg)
	return c
}

// NewCelEnv 创建新的CEL环境
func (c *CustomLib) NewCelEnv() (*cel.Env, error) {
	return cel.NewEnv(cel.Lib(c))
}

// Eval 执行CEL表达式
func Eval(env *cel.Env, expression string, params map[string]any) (ref.Val, error) {
	ast, issues := env.Compile(expression)
	if issues.Err() != nil {
		logger.Error(fmt.Sprintf("CEL编译错误: %s", issues.Err()))
		return nil, issues.Err()
	}

	prg, err := env.Program(ast)
	if err != nil {
		logger.Error(fmt.Sprintf("CEL程序创建错误: %s", err))
		return nil, err
	}

	out, _, err := prg.Eval(params)
	if err != nil {
		logger.Error(fmt.Sprintf("CEL执行错误: %s", err))
		return nil, err
	}

	return out, nil
}

// WriteRuleSetOptions 从YAML配置中添加变量声明
func (c *CustomLib) WriteRuleSetOptions(args yaml.MapSlice) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, v := range args {
		key := v.Key.(string)
		value := v.Value

		var declaration *exprpb.Decl
		switch val := value.(type) {
		case int64:
			declaration = decls.NewVar(key, decls.Int)
		case string:
			if strings.HasPrefix(val, "newReverse") {
				declaration = decls.NewVar(key, decls.NewObjectType("proto.Reverse"))
			} else if strings.HasPrefix(val, "randomInt") {
				declaration = decls.NewVar(key, decls.Int)
			} else {
				declaration = decls.NewVar(key, decls.String)
			}
		case map[string]string:
			declaration = decls.NewVar(key, StrStrMapType)
		default:
			declaration = decls.NewVar(key, decls.String)
		}
		c.envOptions = append(c.envOptions, cel.Declarations(declaration))
	}
}

// WriteRuleFunctionsROptions 注册用于处理r0 || r1规则解析的函数
func (c *CustomLib) WriteRuleFunctionsROptions(funcName string, returnBool bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.envOptions = append(c.envOptions, cel.Function(
		funcName,
		cel.Overload(
			funcName+"_bool",
			[]*cel.Type{},
			cel.BoolType,
			cel.FunctionBinding(func(values ...ref.Val) ref.Val {
				return types.Bool(returnBool)
			}),
		),
	))
}

// UpdateCompileOption 添加变量声明到环境选项
func (c *CustomLib) UpdateCompileOption(varName string, varType *exprpb.Type) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.envOptions = append(c.envOptions, cel.Declarations(decls.NewVar(varName, varType)))
}

// Reset 重置CustomLib实例到初始状态
func (c *CustomLib) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 只重置字段，不替换整个结构体
	c.envOptions = nil
	c.programOptions = nil
	
	// 重新初始化为默认值
	reg := types.NewEmptyRegistry()
	c.envOptions = ReadCompileOptions(reg)
	c.programOptions = ReadProgramOptions(reg)
}

// WriteRuleIsVulOptions 添加漏洞检测函数声明
func (c *CustomLib) WriteRuleIsVulOptions(key string, isVul bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.envOptions = append(c.envOptions, cel.Declarations(decls.NewVar(key+"()", decls.Bool)))
}
