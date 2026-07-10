package handler

import (
	"strings"
	"testing"
)

func TestAdminLogsUseReadableMessagesAndProviderMeta(t *testing.T) {
	handler := readProjectFile(t, "internal", "handler", "handler.go")
	migration := readProjectFile(t, "migrations", "017_repair_log_messages.sql")

	for _, text := range []string{
		"邮箱或密码错误",
		"登录失败",
		"登录成功",
		"已创建生成任务",
		"生成任务已完成",
		"专业绘图任务已提交",
		"智能助手回复完成",
		"智能助手流式回复完成",
	} {
		if !strings.Contains(handler, text) {
			t.Fatalf("handler should write readable log message %q", text)
		}
	}

	for _, text := range []string{
		`taskLogMetaWithProvider(job.Params, taskProvider)`,
		`taskLogMetaWithProvider(req.Params, provider)`,
		`taskLogMetaWithProvider(nil, provider)`,
		`values["providerName"]`,
		`values["model"]`,
	} {
		if !strings.Contains(handler, text) {
			t.Fatalf("task logs should include provider metadata via %s", text)
		}
	}

	if !strings.Contains(migration, "UPDATE login_logs") || !strings.Contains(migration, "UPDATE task_logs") {
		t.Fatal("migration should repair existing login and task log messages")
	}
}
