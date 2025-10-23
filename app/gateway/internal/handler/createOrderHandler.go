package handler

import (
	"net/http"

	"github.com/tsfdsong/tradeengin/app/gateway/internal/logic"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/svc"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func createOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OrderReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewCreateOrderLogic(r.Context(), svcCtx)
		resp, err := l.CreateOrder(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
