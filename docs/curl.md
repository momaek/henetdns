## he.net 登录和 Cookie 行为说明

### 结论（重要）

1. 必须先 `GET https://dns.he.net/` 建立初始 session cookie。
2. 再用同一个 cookie jar `POST` 登录。
3. 登录成功时，响应不一定会返回新的 `Set-Cookie`；登录态可能是绑定在已有 session 上。
4. 所以不能用“登录响应是否有 Set-Cookie”作为唯一成功判据，应该检查页面是否出现 `Logout`/`Welcome`。

### 环境变量

```bash
export HE_EMAIL='your-email'
export HE_PASS='your-password'
```

### 推荐流程（cookie jar）

```bash
set -euo pipefail

COOKIE_JAR="${COOKIE_JAR:-.cache/he.cookies}"
mkdir -p "$(dirname "$COOKIE_JAR")"

# 1) 先访问首页，拿初始 session（通常这一步会有 Set-Cookie）
curl -sS -D /tmp/he.step1.headers -c "$COOKIE_JAR" -b "$COOKIE_JAR" \
  'https://dns.he.net/' \
  -o /tmp/he.step1.body

# 2) 用同一个 session 执行登录（登录响应可能没有新的 Set-Cookie）
curl -sS -L -D /tmp/he.login.headers -c "$COOKIE_JAR" -b "$COOKIE_JAR" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode "email=$HE_EMAIL" \
  --data-urlencode "pass=$HE_PASS" \
  --data-urlencode 'submit=Login!' \
  'https://dns.he.net/' \
  -o /tmp/he.login.body

# 3) 校验是否登录成功
grep -E 'Welcome|Logout' /tmp/he.login.body >/dev/null && echo "login ok" || echo "login failed"
```

### 访问 Zone 页面（复用同一个 cookie jar）

```bash
ZONE_ID=1106664
curl -sS -b "$COOKIE_JAR" \
  "https://dns.he.net/?hosted_dns_zoneid=${ZONE_ID}&menu=edit_zone&hosted_dns_editzone" \
  -o /tmp/he.zone.body
```

### 调试建议

```bash
echo "step1 set-cookie count:"
grep -ci '^Set-Cookie:' /tmp/he.step1.headers || true

echo "login set-cookie count:"
grep -ci '^Set-Cookie:' /tmp/he.login.headers || true
```

如果 `login set-cookie count` 为 `0` 但页面里有 `Welcome/Logout`，这在 he.net 是正常现象。

### 端到端验证（登录后访问 Zone）

下面这个检查更接近真实“点击测试”：
使用同一个 cookie jar 完成 `GET -> POST 登录 -> 请求 zone`，并做 PASS/FAIL 判定。

```bash
set -euo pipefail

COOKIE_JAR="${COOKIE_JAR:-.cache/he.cookies}"
ZONE_ID="${ZONE_ID:-1106664}"
mkdir -p "$(dirname "$COOKIE_JAR")"

# 1) 初始化 session
curl -sS -c "$COOKIE_JAR" -b "$COOKIE_JAR" \
  'https://dns.he.net/' \
  -o /tmp/he.e2e.step1.body

# 2) 登录
curl -sS -L -c "$COOKIE_JAR" -b "$COOKIE_JAR" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode "email=$HE_EMAIL" \
  --data-urlencode "pass=$HE_PASS" \
  --data-urlencode 'submit=Login!' \
  'https://dns.he.net/' \
  -o /tmp/he.e2e.login.body

# 3) 请求 zone 编辑页
curl -sS -c "$COOKIE_JAR" -b "$COOKIE_JAR" \
  "https://dns.he.net/?hosted_dns_zoneid=${ZONE_ID}&menu=edit_zone&hosted_dns_editzone" \
  -o /tmp/he.e2e.zone.body

# 4) PASS/FAIL
if grep -q 'Free DNS Login' /tmp/he.e2e.zone.body; then
  echo 'ZONE_AUTH_CHECK=FAIL(login page returned)'
else
  echo 'ZONE_AUTH_CHECK=PASS(not login page)'
fi

if grep -q 'id="hosted_dns_editzone"' /tmp/he.e2e.zone.body; then
  echo 'ZONE_FORM_CHECK=PASS(edit zone form marker found)'
else
  echo 'ZONE_FORM_CHECK=FAIL(edit zone marker not found)'
fi
```

期望结果：

- `ZONE_AUTH_CHECK=PASS(not login page)`
- `ZONE_FORM_CHECK=PASS(edit zone form marker found)`
