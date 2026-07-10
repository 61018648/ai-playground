UPDATE login_logs
SET message = CASE message
  WHEN '闁喚顔堥幋鏍х槕閻椒绗夊锝団€?' THEN '邮箱或密码错误'
  WHEN '閻ц缍嶆径杈Е' THEN '登录失败'
  WHEN '閻ц缍嶉幋鎰' THEN '登录成功'
  ELSE message
END
WHERE message IN (
  '闁喚顔堥幋鏍х槕閻椒绗夊锝団€?',
  '閻ц缍嶆径杈Е',
  '閻ц缍嶉幋鎰'
);

UPDATE task_logs
SET message = CASE message
  WHEN '閸掓稑缂撻悽鐔稿灇娴犺濮?' THEN '已创建生成任务'
  WHEN '閸楃姳缍呴悽鐔稿灇鐎瑰本鍨?' THEN '生成任务已完成'
  WHEN '涓撲笟缁樺浘浠诲姟宸叉彁浜?' THEN '专业绘图任务已提交'
  WHEN '閺呴缚鍏橀崝鈺傚閸ョ偛顦茬€瑰本鍨?' THEN '智能助手回复完成'
  WHEN '閺呴缚鍏橀崝鈺傚濞翠礁绱￠崶鐐差槻鐎瑰本鍨?' THEN '智能助手流式回复完成'
  ELSE message
END
WHERE message IN (
  '閸掓稑缂撻悽鐔稿灇娴犺濮?',
  '閸楃姳缍呴悽鐔稿灇鐎瑰本鍨?',
  '涓撲笟缁樺浘浠诲姟宸叉彁浜?',
  '閺呴缚鍏橀崝鈺傚閸ョ偛顦茬€瑰本鍨?',
  '閺呴缚鍏橀崝鈺傚濞翠礁绱￠崶鐐差槻鐎瑰本鍨?'
);
