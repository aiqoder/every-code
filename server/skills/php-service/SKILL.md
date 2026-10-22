---
name: PHP 服务
description: 用 PHP 写接口、页面或后台时使用。约束目录结构、输入校验、SQL 写法与错误输出，避免出现注入与静默失败。
---

# PHP 服务

## 工程形态

- 新项目用 Composer 管理依赖，`composer.json` 里固定 PHP 版本要求。
- 入口统一：Web 走 `public/index.php`，命令行走 `bin/` 下的脚本。
- 业务代码放 `src/`，命名空间与目录对应（PSR-4）。
- 配置文件用 `.env` + `vlucas/phpdotenv`，`.env` 不进仓库。

## 必守规则

1. **SQL 一律用预处理语句。** 禁止把变量拼进 SQL 字符串。
2. **输出要转义。** 模板里用 `htmlspecialchars($v, ENT_QUOTES, 'UTF-8')`。
3. **输入要校验。** 不要直接信任 `$_GET` / `$_POST` / JSON body。
4. **错误不能静默。** 生产环境记日志，开发环境显示堆栈；不要 `@` 抑制错误。
5. **密码用 `password_hash` / `password_verify`。** 不要自己写加密。

## 常写片段

```php
// 预处理查询
$stmt = $pdo->prepare('SELECT id, name FROM users WHERE status = ?');
$stmt->execute([$status]);
$rows = $stmt->fetchAll(PDO::FETCH_ASSOC);

// JSON 接口
header('Content-Type: application/json; charset=utf-8');
echo json_encode(['code' => 0, 'data' => $rows], JSON_UNESCAPED_UNICODE);
```

## 验收

- `php -l` 逐文件语法检查通过。
- 主要接口用 `curl` 打一遍，正常与异常分支都有可读返回。
- 页面在浏览器里能打开，无 PHP 警告输出到正文。
