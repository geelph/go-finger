/*
  - Package proto
    @Author: zhizhuo
    @IDE：GoLand
    @File: clone.go
    @Date: 2025/6/14 上午10:00*
*/
package proto

import "bytes"

// CloneRequest 创建 Request 的深拷贝
func CloneRequest(req *Request) *Request {
	if req == nil {
		return nil
	}

	// 创建新的请求对象
	clone := &Request{
		Method:      req.Method,
		ContentType: req.ContentType,
	}

	// 克隆 URL
	if req.Url != nil {
		clone.Url = &UrlType{
			Scheme:   req.Url.Scheme,
			Domain:   req.Url.Domain,
			Host:     req.Url.Host,
			Port:     req.Url.Port,
			Path:     req.Url.Path,
			Query:    req.Url.Query,
			Fragment: req.Url.Fragment,
		}
	}

	// 克隆 Headers
	if req.Headers != nil {
		clone.Headers = make(map[string]string, len(req.Headers))
		for k, v := range req.Headers {
			clone.Headers[k] = v
		}
	}

	// 克隆字节数组
	if req.Body != nil {
		clone.Body = bytes.Clone(req.Body)
	}
	if req.Raw != nil {
		clone.Raw = bytes.Clone(req.Raw)
	}
	if req.RawHeader != nil {
		clone.RawHeader = bytes.Clone(req.RawHeader)
	}

	return clone
}

// CloneResponse 创建 Response 的深拷贝
func CloneResponse(resp *Response) *Response {
	if resp == nil {
		return nil
	}

	// 创建新的响应对象
	clone := &Response{
		Status:      resp.Status,
		ContentType: resp.ContentType,
		Latency:     resp.Latency,
		IconHash:    resp.IconHash,
	}

	// 克隆 URL
	if resp.Url != nil {
		clone.Url = &UrlType{
			Scheme:   resp.Url.Scheme,
			Domain:   resp.Url.Domain,
			Host:     resp.Url.Host,
			Port:     resp.Url.Port,
			Path:     resp.Url.Path,
			Query:    resp.Url.Query,
			Fragment: resp.Url.Fragment,
		}
	}

	// 克隆 Headers
	if resp.Headers != nil {
		clone.Headers = make(map[string]string, len(resp.Headers))
		for k, v := range resp.Headers {
			clone.Headers[k] = v
		}
	}

	// 克隆 Conn
	if resp.Conn != nil {
		clone.Conn = &ConnInfoType{}
		if resp.Conn.Source != nil {
			clone.Conn.Source = &AddrType{
				Transport: resp.Conn.Source.Transport,
				Addr:      resp.Conn.Source.Addr,
				Port:      resp.Conn.Source.Port,
			}
		}
		if resp.Conn.Destination != nil {
			clone.Conn.Destination = &AddrType{
				Transport: resp.Conn.Destination.Transport,
				Addr:      resp.Conn.Destination.Addr,
				Port:      resp.Conn.Destination.Port,
			}
		}
	}

	// 克隆字节数组
	if resp.Body != nil {
		clone.Body = bytes.Clone(resp.Body)
	}
	if resp.Raw != nil {
		clone.Raw = bytes.Clone(resp.Raw)
	}
	if resp.RawHeader != nil {
		clone.RawHeader = bytes.Clone(resp.RawHeader)
	}

	return clone
} 