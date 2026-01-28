package protocol

// InboundBuilder Sing-box inbound 配置构建器
type InboundBuilder struct {
	config map[string]interface{}
}

// NewInboundBuilder 创建 inbound 构建器
func NewInboundBuilder(typ, tag string, port int) *InboundBuilder {
	return &InboundBuilder{
		config: map[string]interface{}{
			"type":        typ,
			"tag":         tag,
			"listen":      "::",
			"listen_port": port,
		},
	}
}

// WithUsers 添加用户配置
func (b *InboundBuilder) WithUsers(users interface{}) *InboundBuilder {
	b.config["users"] = users
	return b
}

// WithTLS 添加 TLS 配置
func (b *InboundBuilder) WithTLS(tls map[string]interface{}) *InboundBuilder {
	b.config["tls"] = tls
	return b
}

// WithTransport 添加传输层配置 (如 gRPC)
func (b *InboundBuilder) WithTransport(transport map[string]interface{}) *InboundBuilder {
	b.config["transport"] = transport
	return b
}

// WithField 添加自定义字段（通用方法）
func (b *InboundBuilder) WithField(key string, value interface{}) *InboundBuilder {
	b.config[key] = value
	return b
}

// Build 构建最终配置
func (b *InboundBuilder) Build() map[string]interface{} {
	return b.config
}

// OutboundBuilder Sing-box outbound 配置构建器
type OutboundBuilder struct {
	config map[string]interface{}
}

// NewOutboundBuilder 创建 outbound 构建器
func NewOutboundBuilder(typ, tag, server string, port int) *OutboundBuilder {
	return &OutboundBuilder{
		config: map[string]interface{}{
			"type":        typ,
			"tag":         tag,
			"server":      server,
			"server_port": port,
		},
	}
}

// WithTLS 添加 TLS 配置
func (o *OutboundBuilder) WithTLS(tls map[string]interface{}) *OutboundBuilder {
	o.config["tls"] = tls
	return o
}

// WithAuth 添加认证信息
func (o *OutboundBuilder) WithAuth(key string, value interface{}) *OutboundBuilder {
	o.config[key] = value
	return o
}

// WithTransport 添加传输层配置
func (o *OutboundBuilder) WithTransport(transport map[string]interface{}) *OutboundBuilder {
	o.config["transport"] = transport
	return o
}

// WithField 添加自定义字段
func (o *OutboundBuilder) WithField(key string, value interface{}) *OutboundBuilder {
	o.config[key] = value
	return o
}

// Build 构建最终配置
func (o *OutboundBuilder) Build() map[string]interface{} {
	return o.config
}
