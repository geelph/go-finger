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
	"gxx/utils/logger"
	"strings"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"gopkg.in/yaml.v2"
)

// 全局CEL环境互斥锁，确保每次只有一个goroutine可以配置环境
var globalCELEnvMutex sync.Mutex

// CustomLib 自定义CEL库结构体
type CustomLib struct {
	envOptions     []cel.EnvOption
	programOptions []cel.ProgramOption
	mutex          sync.RWMutex // 添加锁来保护并发访问
}

// CompileOptions 返回环境选项
func (c *CustomLib) CompileOptions() []cel.EnvOption {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.envOptions
}

// ProgramOptions 返回程序选项
func (c *CustomLib) ProgramOptions() []cel.ProgramOption {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.programOptions
}

// Evaluate 执行CEL表达式并返回结果
func (c *CustomLib) Evaluate(expression string, variables map[string]any) (ref.Val, error) {
	// 全局互斥锁保护 cel.NewEnv() 调用，这是数据竞争的根源
	globalCELEnvMutex.Lock()
	defer globalCELEnvMutex.Unlock()
	
	// 创建一个新环境，在全局锁的保护下
	c.mutex.RLock()
	env, err := cel.NewEnv(cel.Lib(c))
	c.mutex.RUnlock()
	
	if err != nil {
		return nil, fmt.Errorf("创建CEL环境失败: %v", err)
	}
	
	// 复制一份变量映射，避免潜在的并发修改
	varsCopy := make(map[string]any, len(variables))
	for k, v := range variables {
		varsCopy[k] = v
	}
	
	// 编译和评估表达式都在单个线程中完成，不需要额外的并发控制
	return Eval(env, expression, varsCopy)
}

// NewCelEnv 创建新的CEL环境并缓存
func (c *CustomLib) NewCelEnv() (*cel.Env, error) {
	c.mutex.RLock() // 读锁保护对环境选项的访问
	defer c.mutex.RUnlock()
	
	env, err := cel.NewEnv(cel.Lib(c))
	if err != nil {
		return nil, err
	}
	return env, nil
}

// NewCustomLib 创建新的CustomLib实例
func NewCustomLib() *CustomLib {
	c := &CustomLib{
		mutex: sync.RWMutex{},
	}
	reg := types.NewEmptyRegistry()
	c.envOptions = ReadCompileOptions(reg)
	c.programOptions = ReadProgramOptions(reg)
	return c
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
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.envOptions = append(c.envOptions, cel.Declarations(decls.NewVar(varName, varType)))
}

// Reset 重置CustomLib实例到初始状态
func (c *CustomLib) Reset() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.envOptions = append(c.envOptions, cel.Declarations(decls.NewVar(key+"()", decls.Bool)))
}
