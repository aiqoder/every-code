// cpkpack 智能体包打包工具：把开发目录打包为 .cpk 安装包。
//
// 用法：
//
//	go run ./cmd/cpkpack -src ./my-agent -out ./my-agent.cpk
//
// 打包时自动剔除包根 server/ 目录（服务端代码由开发者自行部署），
// 并生成 manifest.sha256 逐文件校验清单。
package main

import (
	"flag"
	"fmt"
	"os"

	"chengzi-agent-code-server/pkg/cpk"
)

func main() {
	src := flag.String("src", "", "包源目录（含 manifest.yaml）")
	out := flag.String("out", "", "输出 .cpk 文件路径")
	flag.Parse()
	if *src == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "用法: cpkpack -src <包源目录> -out <输出.cpk>")
		flag.PrintDefaults()
		os.Exit(2)
	}
	if err := cpk.Pack(*src, *out); err != nil {
		fmt.Fprintln(os.Stderr, "打包失败:", err)
		os.Exit(1)
	}
	// 回读校验
	pkg, err := cpk.Open(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "回读校验失败:", err)
		os.Exit(1)
	}
	defer pkg.Close()
	fmt.Printf("打包成功: %s\n  id: %s\n  name: %s\n  version: %s\n  skills: %v\n  files: %d\n",
		*out, pkg.Manifest.ID, pkg.Manifest.Name, pkg.Manifest.Version, pkg.SkillDirs, len(pkg.Files()))
}
