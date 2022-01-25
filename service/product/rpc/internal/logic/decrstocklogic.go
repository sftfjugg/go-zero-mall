package logic

import (
	"context"
	"database/sql"
	"fmt"

	"mall/service/product/model"
	"mall/service/product/rpc/internal/svc"
	"mall/service/product/rpc/product"

	"github.com/dtm-labs/dtmcli"
	"github.com/dtm-labs/dtmgrpc"
	"github.com/tal-tech/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DecrStockLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDecrStockLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DecrStockLogic {
	return &DecrStockLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DecrStockLogic) DecrStock(in *product.DecrStockRequest) (*product.DecrStockResponse, error) {
	// 获取 RawDB
	db, err := l.svcCtx.ProductModel.RawDB()
	if err != nil {
		return nil, status.Error(codes.Aborted, err.Error())
	}

	// 获取 barrier，用于防止空补偿、空悬挂
	barrier, err := dtmgrpc.BarrierFromGrpc(l.ctx)
	if err != nil {
		return nil, status.Error(codes.Aborted, err.Error())
	}

	if err := barrier.CallWithDB(db, func(tx *sql.Tx) error {
		// 查询产品是否存在
		res, err := l.svcCtx.ProductModel.FindOne(in.Id)
		if err != nil {
			if err == model.ErrNotFound {
				return fmt.Errorf("产品不存在")
			}
			return err
		}

		res.Stock -= in.Num
		if res.Stock < 0 {
			return fmt.Errorf("产品库存不足")
		}

		err = l.svcCtx.ProductModel.TxUpdate(tx, res)
		if err != nil {
			return fmt.Errorf("产品库存处理失败")
		}

		return nil
	}); err != nil {
		return nil, status.Error(codes.Aborted, dtmcli.ResultFailure)
	}

	return &product.DecrStockResponse{}, nil
}