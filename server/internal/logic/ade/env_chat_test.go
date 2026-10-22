package ade

import (
	"strings"
	"testing"
)

func TestEnvCheckPrompt(t *testing.T) {
	got := EnvCheckPrompt("演示", "/tmp/demo", "内部系统", "给财务用", "一个后台")
	for _, want := range []string{
		"【项目】演示",
		"【目录】/tmp/demo",
		"【场景】内部系统",
		"【场景说明】\n给财务用",
		"【项目介绍】\n一个后台",
		"save_env_check",
		`"installed":true`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("提示词缺少 %q\n%s", want, got)
		}
	}
	if strings.Contains(got, "\n\n") {
		t.Fatalf("提示词不应插入空行:\n%s", got)
	}

	bare := EnvCheckPrompt("演示", "/tmp/demo", "", "  ", "")
	if !strings.Contains(bare, "【场景】未选择场景") {
		t.Fatalf("未选场景时应写明: %s", bare)
	}
	if strings.Contains(bare, "场景说明") || strings.Contains(bare, "项目介绍") {
		t.Fatalf("空的场景说明和项目介绍不应出现: %s", bare)
	}
}
