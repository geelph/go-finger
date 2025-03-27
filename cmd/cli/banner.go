/*
  - Package cmd
    @Author: zhizhuo
    @IDE：GoLand
    @File: banner.go
    @Date: 2025/2/20 下午3:39*
*/
package cli

import "fmt"

var version = `1.0.0`

var Banner = fmt.Sprintf(`
 _______  ___  _
/  __\  \/\  \//
| |  _\  / \  / 
| |_///  \ /  \ 
\____/__/\/__/\\

		Version：%s
		Author：zhizhuo
`, version)
