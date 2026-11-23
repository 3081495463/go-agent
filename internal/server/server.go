package server

import (
	"encoding/json"
	"net/http"
	"time"

	"go-agent/internal/agent"
)

// Server 对外暴露的 HTTP API，供前端（Bubble）调用 Agent。
type Server struct {
	Agent *agent.Agent
	Token string
}

// New 创建 HTTP 服务。token 为空时不开启鉴权（仅限本地开发）。
func New(a *agent.Agent, token string) *Server {
	return &Server{Agent: a, Token: token}
}

// Handler 组装路由并套上鉴权中间件。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.Health)
	mux.HandleFunc("/chat", s.Chat)
	return s.auth(mux)
}

// auth 校验 Bearer Token。
func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Token != "" && r.Header.Get("Authorization") != "Bearer "+s.Token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Health 健康检查。
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ChatRequest 聊天请求体。
type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

// Chat 调用 Agent 完成一轮对话。
func (s *Server) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 24*1024)
	defer r.Body.Close()

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	reply, err := s.Agent.Chat(r.Context(), req.SessionID, req.Message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"reply":      reply,
		"session_id": req.SessionID,
	})
}

// Listen 启动 HTTP 服务。
func (s *Server) Listen(addr string) error {
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return httpServer.ListenAndServe()
}
