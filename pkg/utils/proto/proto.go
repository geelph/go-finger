package proto

// 添加的克隆函数用于安全地复制Request和Response对象

// CloneRequest 创建请求对象的深度拷贝
func CloneRequest(src *Request) *Request {
	if src == nil {
		return nil
	}
	
	dst := &Request{
		Method:  src.Method,
		Path:    src.Path,
		Raw:     append([]byte{}, src.Raw...),
		RawPath: append([]byte{}, src.RawPath...),
		Body:    append([]byte{}, src.Body...),
	}
	
	if src.Headers != nil {
		dst.Headers = make(map[string]string, len(src.Headers))
		for k, v := range src.Headers {
			dst.Headers[k] = v
		}
	}
	
	if src.Url != nil {
		dst.Url = &UrlType{
			Scheme:   src.Url.Scheme,
			Domain:   src.Url.Domain,
			Host:     src.Url.Host,
			Port:     src.Url.Port,
			Path:     src.Url.Path,
			Query:    src.Url.Query,
			Fragment: src.Url.Fragment,
		}
	}
	
	return dst
}

// CloneResponse 创建响应对象的深度拷贝
func CloneResponse(src *Response) *Response {
	if src == nil {
		return nil
	}
	
	dst := &Response{
		Status:      src.Status,
		ContentType: src.ContentType,
		Body:        append([]byte{}, src.Body...),
		Raw:         append([]byte{}, src.Raw...),
		RawHeader:   append([]byte{}, src.RawHeader...),
		Latency:     src.Latency,
	}
	
	if src.Headers != nil {
		dst.Headers = make(map[string]string, len(src.Headers))
		for k, v := range src.Headers {
			dst.Headers[k] = v
		}
	}
	
	if src.Url != nil {
		dst.Url = &UrlType{
			Scheme:   src.Url.Scheme,
			Domain:   src.Url.Domain,
			Host:     src.Url.Host,
			Port:     src.Url.Port,
			Path:     src.Url.Path,
			Query:    src.Url.Query,
			Fragment: src.Url.Fragment,
		}
	}
	
	return dst
} 