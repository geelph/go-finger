/*
  - Package cel
    @Author: zhizhuo
    @IDE：GoLand
    @File: celcompile.go
    @Date: 2025/2/7 上午8:57*
*/
package cel

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"gxx/utils/pkg/proto"
)

// StrStrMapType 定义了一个键和值都为字符串的映射类型
var StrStrMapType = decls.NewMapType(decls.String, decls.String)

// NewEnvOptions 定义了默认的 CEL 环境选项
var NewEnvOptions = []cel.EnvOption{
	cel.Container("proto"),
	cel.Types(
		&proto.UrlType{},
		&proto.Request{},
		&proto.Response{},
		&proto.Reverse{},
		StrStrMapType,
	),
	cel.Declarations(
		decls.NewVar("request", decls.NewObjectType("proto.Request")),
		decls.NewVar("response", decls.NewObjectType("proto.Response")),
		// 字符串处理函数
		decls.NewFunction("icontains",
			decls.NewInstanceOverload("string_icontains_string",
				[]*exprpb.Type{decls.String, decls.String},
				decls.Bool)),
		decls.NewFunction("substr",
			decls.NewOverload("substr_string_int_int",
				[]*exprpb.Type{decls.String, decls.Int, decls.Int},
				decls.String)),
		decls.NewFunction("replaceAll",
			decls.NewOverload("replaceAll_string_string_string",
				[]*exprpb.Type{decls.String, decls.String, decls.String},
				decls.String)),
		decls.NewFunction("printable",
			decls.NewOverload("printable_string",
				[]*exprpb.Type{decls.String},
				decls.String)),
		decls.NewFunction("toUintString",
			decls.NewOverload("toUintString_string_string",
				[]*exprpb.Type{decls.String, decls.String},
				decls.String)),
		// 字节处理函数
		decls.NewFunction("bcontains",
			decls.NewInstanceOverload("bytes_bcontains_bytes",
				[]*exprpb.Type{decls.Bytes, decls.Bytes},
				decls.Bool)),
		decls.NewFunction("ibcontains",
			decls.NewInstanceOverload("bytes_ibcontains_bytes",
				[]*exprpb.Type{decls.Bytes, decls.Bytes},
				decls.Bool)),
		decls.NewFunction("bstartsWith",
			decls.NewInstanceOverload("bytes_bstartsWith_bytes",
				[]*exprpb.Type{decls.Bytes, decls.Bytes},
				decls.Bool)),
		// 编码函数
		decls.NewFunction("md5",
			decls.NewOverload("md5_string",
				[]*exprpb.Type{decls.String},
				decls.String)),
		decls.NewFunction("base64",
			decls.NewOverload("base64_string",
				[]*exprpb.Type{decls.String},
				decls.String)),
		decls.NewFunction("base64",
			decls.NewOverload("base64_bytes",
				[]*exprpb.Type{decls.Bytes},
				decls.String)),
		decls.NewFunction("base64Decode",
			decls.NewOverload("base64Decode_string",
				[]*exprpb.Type{decls.String},
				decls.String)),
		decls.NewFunction("base64Decode",
			decls.NewOverload("base64Decode_bytes",
				[]*exprpb.Type{decls.Bytes},
				decls.String)),
		decls.NewFunction("urlencode",
			decls.NewOverload("urlencode_string",
				[]*exprpb.Type{decls.String},
				decls.String)),
		decls.NewFunction("urlencode",
			decls.NewOverload("urlencode_bytes",
				[]*exprpb.Type{decls.Bytes},
				decls.String)),
		decls.NewFunction("urldecode",
			decls.NewOverload("urldecode_string",
				[]*exprpb.Type{decls.String},
				decls.String)),
		decls.NewFunction("urldecode",
			decls.NewOverload("urldecode_bytes",
				[]*exprpb.Type{decls.Bytes},
				decls.String)),
		decls.NewFunction("faviconHash",
			decls.NewOverload("faviconHash_stringOrBytes",
				[]*exprpb.Type{decls.Any},
				decls.Int)),
		decls.NewFunction("hexdecode",
			decls.NewOverload("hexdecode_string",
				[]*exprpb.Type{decls.String},
				decls.String)),
		// 随机数生成函数
		decls.NewFunction("randomInt",
			decls.NewOverload("randomInt_int_int",
				[]*exprpb.Type{decls.Int, decls.Int},
				decls.Int)),
		decls.NewFunction("randomLowercase",
			decls.NewOverload("randomLowercase_int",
				[]*exprpb.Type{decls.Int},
				decls.String)),
		// 正则表达式函数
		decls.NewFunction("submatch",
			decls.NewInstanceOverload("string_submatch_string",
				[]*exprpb.Type{decls.String, decls.String},
				StrStrMapType)),
		decls.NewFunction("bsubmatch",
			decls.NewInstanceOverload("string_bsubmatch_bytes",
				[]*exprpb.Type{decls.String, decls.Bytes},
				StrStrMapType)),
		decls.NewFunction("bmatches",
			decls.NewInstanceOverload("string_bmatches_bytes",
				[]*exprpb.Type{decls.String, decls.Bytes},
				decls.Bool)),
		// 反向函数
		decls.NewFunction("wait",
			decls.NewInstanceOverload("reverse_wait_int",
				[]*exprpb.Type{decls.Any, decls.Int},
				decls.Bool)),
		decls.NewFunction("jndi",
			decls.NewInstanceOverload("reverse_jndi_int",
				[]*exprpb.Type{decls.Any, decls.Int},
				decls.Bool)),
		// 其他函数
		decls.NewFunction("sleep",
			decls.NewOverload("sleep_int", []*exprpb.Type{decls.Int},
				decls.Null)),
		// 年份函数
		decls.NewFunction("year",
			decls.NewOverload("year_string", []*exprpb.Type{decls.Int},
				decls.String)),
		decls.NewFunction("shortyear",
			decls.NewOverload("shortyear_string", []*exprpb.Type{decls.Int},
				decls.String)),
		decls.NewFunction("month",
			decls.NewOverload("month_string", []*exprpb.Type{decls.Int},
				decls.String)),
		decls.NewFunction("day",
			decls.NewOverload("day_string", []*exprpb.Type{decls.Int},
				decls.String)),
		decls.NewFunction("timestamp_second",
			decls.NewOverload("timestamp_second_string", []*exprpb.Type{decls.Int},
				decls.String)),
	),
}

// ReadCompileOptions 返回包含自定义类型提供者和适配器的 CEL 环境选项
func ReadCompileOptions(reg ref.TypeRegistry) []cel.EnvOption {
	allEnvOptions := []cel.EnvOption{
		cel.CustomTypeAdapter(reg),
		cel.CustomTypeProvider(reg),
	}
	allEnvOptions = append(allEnvOptions, NewEnvOptions...)
	return allEnvOptions
}
