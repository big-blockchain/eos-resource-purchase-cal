# eos-resource-purchase-cal
calculate eos resource fee

## Info
this rep complete eosio cpu、net、ram resource fee calculate

## Calculate cpu and net fee
```
    // get user resource info
	cpuNetFrac := CpuNetFrac{}
	accountInfo, err := CtxSvc.ChainRpc.GetAccount(context.Background(), &chainserviceclient.GetAccountReq{Account: account})
	if err != nil {
		return cpuNetFrac, err
	}
	
    //get eosio mainnet powup.state info
	reqGetState := chainserviceclient.TabletRowRequest{
		Json:  true,
		Code:  "eosio",
		Scope: "",
		Table: "powup.state",
	}
	res, err2 := CtxSvc.ChainRpc.GetTableRows(context.Background(), &reqGetState)
	if err2 != nil {
		return cpuNetFrac, err
	}

	eosResourceUtils := chainlib.New(res, accountInfo)
	sample := eosResourceUtils.Sample

	cpuWeightQu, _ := eosgo.NewAssetFromString(cpuWeight)
	//cal cpu us
	cpuUs := eosResourceUtils.WeightToUs(sample.Cpu, int64(cpuWeightQu.Amount))
	//cal cpu frac by cpu us
	cpuFrac := eosResourceUtils.FracByUs(sample, cpuUs)

	netWeightQu, _ := eosgo.NewAssetFromString(netWeight)
	netBytes := eosResourceUtils.WeightToBytes(sample.Net, int64(netWeightQu.Amount))
	netFrac := eosResourceUtils.FracByBytes(sample, netBytes)
	
	//cal cpu and net eos fee
	price1 := eosResourceUtils.PricePerUs(sample, cpuUs)
	price2 := eosResourceUtils.PricePerByte(sample, netBytes)
```

## Calculate Ram Fee
```
    //get market info
    reqGetState := chainserviceclient.TabletRowRequest{
		Json:  true,
		Code:  "eosio",
		Scope: "eosio",
		Table: "rammarket",
	}
	res, err2 := l.svcCtx.ChainRpc.GetTableRows(l.ctx, &reqGetState)
	if err2 != nil {
		return &types.CalculateRamPriceRes{}, status.Errorf(500, err2.Error())
	}
	//cal ram fee
	result := chainlib.CalcRamPricePerBytes(res.Rows, float64(req.RamBytes))
```
