package tasks

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RedisClient struct {
	mu        sync.Mutex
	conn      net.Conn
	reader    *bufio.Reader
	host      string
	port      int
	password  string
	user      string
	prefix    string
	ttl       time.Duration
	connected bool
}

const reconnectCooldown = 5 * time.Second

var lastReconnectFail time.Time

func redisNew(host string, port int, password, user, prefix string, ttl int) *RedisClient {
	c := &RedisClient{
		host:     host,
		port:     port,
		password: password,
		user:     user,
		prefix:   prefix,
		ttl:      time.Duration(ttl) * time.Minute,
	}
	c.reconnect()
	return c
}

func (c *RedisClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// reconnect 尝试重建 TCP 连接并认证，冷却期内不重试
func (c *RedisClient) reconnect() {
	if time.Since(lastReconnectFail) < reconnectCooldown {
		return
	}
	addr := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		lastReconnectFail = time.Now()
		return
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.connected = true
	if c.password != "" {
		user := c.user
		pass := c.password
		// 直接发送 AUTH，不走 cmd 避免递归重连
		authCmd := fmt.Sprintf("*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(pass), pass)
		if user != "" {
			authCmd = fmt.Sprintf("*3\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
				len(user), user, len(pass), pass)
		}
		if _, wErr := conn.Write([]byte(authCmd)); wErr != nil {
			c.connected = false
			lastReconnectFail = time.Now()
			conn.Close()
			return
		}
		line, rErr := c.reader.ReadString('\n')
		if rErr != nil || len(line) > 0 && line[0] == '-' {
			c.connected = false
			lastReconnectFail = time.Now()
			conn.Close()
			return
		}
	}
}

func (c *RedisClient) Set(key, value string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return false
	}
	return c.cmd("SET", c.prefix+key, value, "EX", strconv.Itoa(int(c.ttl.Seconds()))) == nil
}

func (c *RedisClient) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return "", false
	}
	result, err := c.cmdReply("GET", c.prefix+key)
	return result, err == nil && result != ""
}

func (c *RedisClient) Del(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return false
	}
	return c.cmd("DEL", c.prefix+key) == nil
}

// SetNX 设置 key-value，仅当 key 不存在时才设置
// 返回 true 表示设置成功（key 不存在），false 表示 key 已存在
func (c *RedisClient) SetNX(key, value string, ttl time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		// Redis 不可用时返回 true，允许继续使用（不去重）
		return true
	}
	seconds := int(ttl.Seconds())
	if seconds <= 0 {
		seconds = int(c.ttl.Seconds())
	}
	err := c.cmd("SET", c.prefix+key, value, "NX", "EX", strconv.Itoa(seconds))
	return err == nil
}

func (c *RedisClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.connected = false
	}
}

func (c *RedisClient) cmd(parts ...string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d\r\n", len(parts)))
	for _, p := range parts {
		sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(p), p))
	}
	data := []byte(sb.String())
	if _, err := c.conn.Write(data); err != nil {
		c.connected = false
		c.reconnect()
		if c.connected {
			_, err = c.conn.Write(data)
		}
		if err != nil {
			return err
		}
	}
	line, err := c.reader.ReadString('\n')
	if err != nil {
		c.connected = false
		c.reconnect()
		if c.connected {
			line, err = c.reader.ReadString('\n')
		}
		if err != nil {
			return err
		}
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) > 0 && line[0] == '-' {
		return fmt.Errorf("redis: %s", line[1:])
	}
	return nil
}

func (c *RedisClient) cmdReply(parts ...string) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d\r\n", len(parts)))
	for _, p := range parts {
		sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(p), p))
	}
	data := []byte(sb.String())
	if _, err := c.conn.Write(data); err != nil {
		c.connected = false
		c.reconnect()
		if c.connected {
			_, err = c.conn.Write(data)
		}
		if err != nil {
			return "", err
		}
	}

	line, err := c.reader.ReadString('\n')
	if err != nil {
		c.connected = false
		c.reconnect()
		if c.connected {
			line, err = c.reader.ReadString('\n')
		}
		if err != nil {
			return "", err
		}
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return "", fmt.Errorf("redis: empty reply")
	}
	switch line[0] {
	case '+', ':':
		return line[1:], nil
	case '-':
		return "", fmt.Errorf("redis: %s", line[1:])
	case '$':
		length, _ := strconv.Atoi(line[1:])
		if length == -1 {
			return "", nil
		}
		buf := make([]byte, length+2)
		c.reader.Read(buf)
		return string(buf[:length]), nil
	default:
		return line, nil
	}
}
