这是登录的后台逻辑请参考
```go
package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"gateway/setting"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// 定义一个jwt对象
type JWT struct {
	// 声明签名信息
	SigningKey []byte
}

// 自定义有效载荷(这里采用自定义的Name和Email作为有效载荷的一部分)
type CustomClaims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	// StandardClaims结构体实现了Claims接口(Valid()函数)
	jwt.StandardClaims
}

// 构造用户表
type User struct {
	Id        int32  `gorm:"AUTO_INCREMENT"`
	Name      string `json:"username"`
	Pwd       string `json:"password"`
	Phone     int64  `gorm:"DEFAULT:0"`
	Email     string `gorm:"type:varchar(20);unique_index;"`
	CreatedAt *time.Time
	UpdateTAt *time.Time
}

// LoginReq请求参数
type LoginReq struct {
	Name  string `json:"username"`
	Pwd   string `json:"password"` // 前端传来的 SHA256(password + nonce)
	Nonce string `json:"nonce"`    // 前端传来的一次性随机串
}

// 登陆结果
type LoginResultTemplate struct {
	Token       string                        `json:"token"`
	Name        string                        `json:"username"`
	Permissions []setting.PermissionsTemplate `json:"permissions"`
}

// ==========================
// 登录爆破防护逻辑（新增）
// ==========================

// 允许最大失败次数
const MaxLoginFailCount = 10

// 达到最大失败次数后的封锁时长
const LoginBlockDuration = 3 * time.Minute

// 记录登录失败次数
type LoginFailInfo struct {
	Count      int
	LastFailAt time.Time
	BlockUntil time.Time
}

var loginFailMap = make(map[string]*LoginFailInfo)

var (
	TokenExpired error = errors.New("Token is expired")

	LoginResult LoginResultTemplate
)

// 初始化jwt对象
func NewJWT() *JWT {
	return &JWT{
		[]byte("GATEWAY"),
	}
}

// 调用jwt-go库生成token
// 指定编码的算法为jwt.SigningMethodHS256
func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	// https://gowalker.org/github.com/dgrijalva/jwt-go#Token
	// 返回一个token的结构体指针
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

// token解码
func (j *JWT) ParserToken(tokenString string) (*CustomClaims, error) {
	// https://gowalker.org/github.com/dgrijalva/jwt-go#ParseWithClaims
	// 输入用户自定义的Claims结构体对象,token,以及自定义函数来解析token字符串为jwt的Token结构体指针
	// Keyfunc是匿名函数类型: type Keyfunc func(*Token) (interface{}, error)
	// func ParseWithClaims(tokenString string, claims Claims, keyFunc Keyfunc) (*Token, error) {}
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	})

	if err != nil {
		// https://gowalker.org/github.com/dgrijalva/jwt-go#ValidationError
		// jwt.ValidationError 是一个无效token的错误结构
		if ve, ok := err.(*jwt.ValidationError); ok {
			// ValidationErrorMalformed是一个uint常量，表示token不可用
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, fmt.Errorf("token不可用")
				// ValidationErrorExpired表示Token过期
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, fmt.Errorf("token过期")
				// ValidationErrorNotValidYet表示无效token
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, fmt.Errorf("无效的token")
			} else {
				return nil, fmt.Errorf("token不可用")
			}

		}
	}

	// 将token中的claims信息解析出来并断言成用户自定义的有效载荷结构
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("token无效")

}

// 定义一个JWTAuth的中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 通过http header中的token解析来认证
		token := c.Request.Header.Get("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "1",
				"message": "请求未携带token，无权限访问",
				"data":    "",
			})
			c.Abort()
			return
		}
		// 初始化一个JWT对象实例，并根据结构体方法来解析token
		j := NewJWT()
		// 解析token中包含的相关信息(有效载荷)
		claims, err := j.ParserToken(token)
		if err != nil {
			// token过期
			if err.Error() == "token不可用" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    "1",
					"message": "token不可用",
					"data":    "",
				})
				c.Abort()
				return
			} else if err.Error() == "token过期" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    "1",
					"message": "登录已经过期，请重新登录",
					"data":    "",
				})
				c.Abort()
				return
			}

			setting.ZAPS.Errorf("gin解析token错误 %v", err)

			// 其他错误
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "1",
				"message": fmt.Sprintf("gin解析token错误 %v", err.Error()),
				"data":    "",
			})
			c.Abort()
			return
		}

		// 将解析后的有效载荷claims重新写入gin.Context引用对象中
		c.Set("claims", claims)
	}
}

// LoginCheck验证
// LoginCheck验证
func LoginCheck(login LoginReq) (bool, User, error) {
	userData := User{}
	userExist := false
	if !ValidateAndConsumeNonce(login.Nonce) {
		return false, userData, fmt.Errorf("加密已过期,请再次登录")
	}
	// 遍历配置用户
	for _, v := range setting.PolicyWeb {
		if v.Role == login.Name {
			userExist = true
			// 计算数据库密码 + nonce 的 SHA256
			hashBytes := sha256.Sum256([]byte(v.Password + login.Nonce))
			serverHash := hex.EncodeToString(hashBytes[:])

			if serverHash != login.Pwd {
				return false, userData, fmt.Errorf("登陆信息有误")
			}

			userData.Name = v.Role
			userData.Email = "" // 可根据需要填写
			return true, userData, nil
		}
	}

	if !userExist {
		return false, userData, fmt.Errorf("用户不存在")
	}

	return false, userData, fmt.Errorf("未知错误")
}

// token生成器
// md 为上面定义好的middleware中间件
func GenerateToken(c *gin.Context, user User) {
	// 构造SignKey: 签名和解签名需要使用一个值
	j := NewJWT()

	// 构造用户claims信息(负荷)
	claims := CustomClaims{
		user.Name,
		user.Email,
		jwt.StandardClaims{
			NotBefore: time.Now().Unix(),           // 签名生效时间
			ExpiresAt: time.Now().Unix() + 3600*96, // 签名过期时间
			Issuer:    "HNE",                       // 签名颁发者
		},
	}

	// 根据claims生成token对象
	token, err := j.CreateToken(claims)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    "1",
			"message": err.Error(),
			"data":    "",
		})
		return
	}

	for _, v := range setting.PolicyWeb {
		if v.Role == claims.Name {
			LoginResult.Permissions = v.Policy
		}
	}

	data := LoginResultTemplate{
		Name:        user.Name,
		Token:       token,
		Permissions: LoginResult.Permissions,
	}
	LoginResult = data
	c.JSON(http.StatusOK, gin.H{
		"code":    "0",
		"message": "登录成功",
		"data":    data,
	})
	return

}

// 获取客户端真实 IP
func GetClientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		ip = c.RemoteIP()
	}
	if ip == "" {
		ip = "unknown"
	}
	return ip
}

// 检查当前 IP 是否被封锁
func IsIPBlocked(ip string) (bool, time.Duration) {
	info, exists := loginFailMap[ip]
	if !exists {
		return false, 0
	}

	if time.Now().Before(info.BlockUntil) {
		return true, time.Until(info.BlockUntil)
	}

	// 自动解除封锁
	return false, 0
}

// 记录失败
func AddLoginFail(ip string) {
	info, exists := loginFailMap[ip]
	if !exists {
		loginFailMap[ip] = &LoginFailInfo{
			Count:      1,
			LastFailAt: time.Now(),
			BlockUntil: time.Time{},
		}
		return
	}

	info.Count++
	info.LastFailAt = time.Now()

	if info.Count >= MaxLoginFailCount {
		info.BlockUntil = time.Now().Add(LoginBlockDuration)
	}
}

// 登录成功后清除
func ClearLoginFail(ip string) {
	delete(loginFailMap, ip)
}
```

随机数的逻辑
```go
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// =====================
//
//	nonce 全局存储
//
// =====================
var (
	nonceStore sync.Map
	nonceMax   = 100000 // 最大允许缓存的 nonce 数量，避免内存膨胀
)

// =====================
//
//	IP 限流器（登录、防刷 nonce）
//
// =====================
var nonceLimiters sync.Map

func GetLimiter(ip string) *rate.Limiter {
	if l, ok := nonceLimiters.Load(ip); ok {
		return l.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(2, 5) // 每秒 5 次、突发 10 次
	nonceLimiters.Store(ip, limiter)
	return limiter
}

// =====================
//
//	生成 nonce（一次性随机值，有效期 2 分钟）
//
// =====================
func GenerateNonce() (string, error) {
	// 防止 DoS：nonceStore 超过上限直接拒绝生成
	size := 0
	nonceStore.Range(func(_, _ any) bool {
		size++
		return true
	})
	if size > nonceMax {
		return "", nil
	}

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	nonce := hex.EncodeToString(b)
	nonceStore.Store(nonce, time.Now().Add(2*time.Minute)) // 保存过期时间

	return nonce, nil
}

// =====================
//
//	验证并消耗 nonce
//
// =====================
func ValidateAndConsumeNonce(nonce string) bool {
	v, ok := nonceStore.Load(nonce)
	if !ok {
		return false
	}

	expire := v.(time.Time)
	if time.Now().After(expire) {
		nonceStore.Delete(nonce)
		return false
	}

	// 验证通过后立即删除，防重放攻击
	nonceStore.Delete(nonce)
	return true
}

// =====================
//
//	启动后台 GC 清理过期 nonce
//
// =====================
func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			nonceStore.Range(func(key, value any) bool {
				expire := value.(time.Time)
				if time.Now().After(expire) {
					nonceStore.Delete(key)
				}
				return true
			})
		}
	}()
}
```