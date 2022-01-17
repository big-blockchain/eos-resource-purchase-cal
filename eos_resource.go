package chainlib

import (
	"github.com/big-blockchain/utils/utils"
	eosgo "github.com/eoscanada/eos-go"
	"gitlab.utu.one/ariginal/backend/protos/module-chain/chainserviceclient"
	"math"
	"strconv"
	"strings"
	"time"
)

type EosResourceUtils struct {
	RowsData StateRow
	Sample   SampleUsage
}

type StateRow struct {
	Version       int64     `json:"version"`
	Net           StateData `json:"net"`
	Cpu           StateData `json:"cpu"`
	PowerupDays   int64     `json:"powerup_days"`
	MinPowerupFee string    `json:"min_powerup_fee"`
}

type StateData struct {
	Version              int64  `json:"version"`
	Weight               string `json:"weight"`
	WeightRatio          string `json:"weight_ratio"`
	AssumedStakeWeight   string `json:"assumed_stake_weight"`
	InitialWeightRatio   string `json:"initial_weight_ratio"`
	TargetWeightRatio    string `json:"target_weight_ratio"`
	InitialTimestamp     string `json:"initial_timestamp"`
	TargetTimestamp      string `json:"target_timestamp"`
	Exponent             string `json:"exponent"`
	DecaySecs            int64  `json:"decay_secs"`
	MinPrice             string `json:"min_price"`
	MaxPrice             string `json:"max_price"`
	Utilization          string `json:"utilization"`
	AdjustedUtilization  string `json:"adjusted_utilization"`
	UtilizationTimestamp string `json:"utilization_timestamp"`
}

type SampleUsage struct {
	Cpu float64 `json:"cpu"`
	Net float64 `json:"net"`
}

func New(powerUpState *chainserviceclient.TableRowResponse, accountInfo *chainserviceclient.GetAccountRes) *EosResourceUtils {
	l := EosResourceUtils{}
	rows := make([]StateRow, 0)
	utils.JsonUtils{}.JsonDecode(powerUpState.Rows, &rows)
	l.RowsData = rows[0]

	us := accountInfo.CpuLimit.Max * 1000000
	if accountInfo.CpuLimit.Max == 0 {
		us = 1
	}
	bytes := accountInfo.NetLimit.Max * 1000000
	if accountInfo.NetLimit.Max == 0 {
		bytes = 1000000
	}

	cpuWeight := accountInfo.CpuWeight
	netWeight := accountInfo.NetWeight

	cpuV := float64(us) / float64(cpuWeight)
	cpuMod := math.Mod(float64(us), float64(cpuWeight))
	if cpuMod > 0 {
		cpuV = cpuV - 1
	}

	netV := float64(bytes) / float64(netWeight)
	netMod := math.Mod(float64(bytes), float64(netWeight))
	if netMod > 0 {
		netV = netV - 1
	}
	sample := SampleUsage{}
	sample.Net = math.Abs(netV)
	sample.Cpu = math.Abs(cpuV)
	l.Sample = sample
	return &l
}

func (l *EosResourceUtils) PricePerUs(sample SampleUsage, us float64) float64 {
	frac := l.FracByUs(sample, us)
	weight, _ := strconv.ParseFloat(l.RowsData.Cpu.Weight, 64)
	utilizationIncrease := l.UtilizationIncrease(weight, frac)
	adjustedUtilization := l.DetermineAdjustedUtilization(l.RowsData.Cpu)
	fee := l.Fee(l.RowsData.Cpu, utilizationIncrease, adjustedUtilization)
	precision := math.Pow(10, 4)
	value := math.Ceil(fee*precision) / precision
	return value
}

func (l *EosResourceUtils) PricePerByte(sample SampleUsage, bytes float64) float64 {
	frac := l.FracByBytes(sample, bytes)
	weight, _ := strconv.ParseFloat(l.RowsData.Net.Weight, 64)
	utilizationIncrease := l.UtilizationIncrease(weight, frac)
	adjustedUtilization := l.DetermineAdjustedUtilization(l.RowsData.Net)
	fee := l.Fee(l.RowsData.Net, utilizationIncrease, adjustedUtilization)
	precision := math.Pow(10, 4)
	value := math.Ceil(fee*precision) / precision
	return value
}

func (l *EosResourceUtils) FracByBytes(sample SampleUsage, bytes float64) float64 {
	weight, _ := strconv.ParseFloat(l.RowsData.Net.Weight, 64)
	frac := l.BytesToWeight(sample.Net, bytes) / weight
	return math.Floor(frac * math.Pow(10, 15))
}

func (l *EosResourceUtils) FracByUs(sample SampleUsage, us float64) float64 {
	weight, _ := strconv.ParseFloat(l.RowsData.Cpu.Weight, 64)
	frac := l.UsToWeight(sample.Cpu, us) / weight
	return math.Floor(frac * math.Pow(10, 15))
}

func (l *EosResourceUtils) WeightToUs(sample float64, weight int64) float64 {
	return math.Ceil(sample * float64(weight) / 1000000)
}

func (l *EosResourceUtils) WeightToBytes(sample float64, weight int64) float64 {
	return math.Ceil(sample * float64(weight) / 1000000)
}

func (l *EosResourceUtils) UsToWeight(sample, us float64) float64 {
	return math.Floor(us / sample * 1000000)
}

func (l *EosResourceUtils) BytesToWeight(sample, bytes float64) float64 {
	return math.Floor(bytes / sample * 1000000)
}

func (l *EosResourceUtils) UtilizationIncrease(weight, frac float64) float64 {
	utilizationIncrease := (weight * frac) / math.Pow(10, 15)
	return math.Ceil(utilizationIncrease)
}

func (l *EosResourceUtils) DetermineAdjustedUtilization(data StateData) float64 {
	decaySecs := data.DecaySecs
	utilization, _ := strconv.ParseFloat(data.Utilization, 64)
	utilizationTimestamp, _ := time.Parse("2006-01-02T15:04:05", data.UtilizationTimestamp) //TODO 测试转换
	adjustedUtilization, _ := strconv.ParseFloat(data.AdjustedUtilization, 64)
	if utilization < adjustedUtilization {
		now := time.Now().UTC().Unix()
		diff := adjustedUtilization - utilization
		delta := diff * math.Exp(float64(utilizationTimestamp.Unix()-now)/float64(decaySecs))
		delta1 := math.Min(math.Max(delta, 0), diff)
		adjustedUtilization = utilization + delta1
	}
	return adjustedUtilization
}

func (l *EosResourceUtils) Fee(data StateData, utilizationIncrease, adjustedUtilization float64) float64 {
	utilization, _ := strconv.ParseFloat(data.Utilization, 64)
	weight, _ := strconv.ParseFloat(data.Weight, 64)
	startUtilization := utilization
	endUtilization := startUtilization + utilizationIncrease
	var fee float64
	if startUtilization < adjustedUtilization {
		fee += l.PriceFunction(data, utilization) * (math.Min(utilizationIncrease, (adjustedUtilization - startUtilization))) / weight
		startUtilization = adjustedUtilization
	}
	if startUtilization < endUtilization {
		fee += l.PriceFunctionDelta(data, startUtilization, endUtilization)
	}

	return fee
}

func (l *EosResourceUtils) PriceFunction(data StateData, utilization float64) float64 {
	exponent, _ := strconv.ParseFloat(data.Exponent, 64)
	weight, _ := strconv.ParseInt(data.Weight, 10, 64)

	max, _ := eosgo.NewAssetFromString(data.MaxPrice)
	maxPrice := float64(max.Amount) / 10000
	min, _ := eosgo.NewAssetFromString(data.MinPrice)
	minPrice := float64(min.Amount) / 10000

	price := minPrice
	newExponent := exponent - 1.0
	if newExponent <= 0.0 {
		return maxPrice
	} else {
		price += (maxPrice - minPrice) * math.Pow(utilization/float64(weight), newExponent)
	}
	return price
}

func (l *EosResourceUtils) PriceFunctionDelta(data StateData, startUtilization, endUtilization float64) float64 {
	exponent, _ := strconv.ParseFloat(data.Exponent, 64)
	weight, _ := strconv.ParseInt(data.Weight, 10, 64)
	max, _ := eosgo.NewAssetFromString(data.MaxPrice)
	maxPrice := float64(max.Amount)
	min, _ := eosgo.NewAssetFromString(data.MinPrice)
	minPrice := float64(min.Amount)
	coefficient := (maxPrice - minPrice) / exponent
	startU := startUtilization / float64(weight)
	endU := endUtilization / float64(weight)
	delta := minPrice*endU - minPrice*startU + coefficient*math.Pow(endU, exponent) - coefficient*math.Pow(startU, exponent)
	return delta
}

type RamMarketRow struct {
	Supply string `json:"supply"`
	Base   struct {
		Balance string `json:"balance"`
		Weight  string `json:"weight"`
	} `json:"base"`
	Quote struct {
		Balance string `json:"balance"`
		Weight  string `json:"weight"`
	} `json:"quote"`
}

func RamGetInput(base, quote, bytes float64) float64 {
	result := (quote * bytes) / (base - bytes)
	if result < 0 {
		return 0
	}
	return result
}

func CalcRamPricePerBytes(ramMarketRowString string, bytes float64) float64 {
	ramMarketRows := make([]RamMarketRow, 0)
	err := utils.JsonUtils{}.JsonDecode(ramMarketRowString, &ramMarketRows)
	if len(ramMarketRows) == 0 || err != nil {
		return 0
	}
	quoteBalance, _ := eosgo.NewAssetFromString(ramMarketRows[0].Quote.Balance)
	quote := float64(quoteBalance.Amount) / 10000
	base, _ := strconv.ParseFloat(strings.ReplaceAll(ramMarketRows[0].Base.Balance, " RAM", ""), 64)
	result := RamGetInput(base, quote, bytes)
	return result
}
