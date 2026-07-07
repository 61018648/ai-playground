# 安装初始化说明

本项目推荐使用安装命令初始化空数据库，而不是导出现有数据库数据。安装命令会自动完成：

- 创建 `image-ai` 数据库（如果不存在）
- 执行 `backend/migrations/*.sql`
- 创建或更新初始管理员

## 默认初始化信息

当前系统使用邮箱登录，所以默认管理员信息为：

```txt
邮箱：admin@example.com
昵称：admin
密码：123456
```

## 一键安装

进入后端目录：

```powershell
cd E:\my-ai-image-app\backend
```

执行安装：

```powershell
$env:DATABASE_URL="postgres://postgres:123456@127.0.0.1:5432/image-ai?sslmode=disable"
go run ./cmd/dbinit
```

安装完成后启动后端：

```powershell
go run ./cmd/api
```

## 自定义初始信息

可以在安装前设置环境变量：

```powershell
$env:DATABASE_URL="postgres://postgres:123456@127.0.0.1:5432/image-ai?sslmode=disable"
$env:INSTALL_ADMIN_EMAIL="admin@your-domain.com"
$env:INSTALL_ADMIN_PASSWORD="123456"
$env:INSTALL_ADMIN_NICKNAME="admin"
go run ./cmd/dbinit
```

默认情况下，重复执行安装命令会把该管理员密码重置为 `INSTALL_ADMIN_PASSWORD`。如果只想补迁移和确保管理员权限，不想重置密码：

```powershell
$env:INSTALL_SKIP_ADMIN_PASSWORD_RESET="true"
go run ./cmd/dbinit
```

## 仍然只执行迁移

如果不需要创建初始管理员，可以继续使用原迁移命令：

```powershell
go run ./cmd/migrate
```
