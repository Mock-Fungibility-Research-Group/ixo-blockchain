package project

import (
	"encoding/hex"
	"fmt"

	"github.com/tendermint/tmlibs/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	ixo "github.com/ixofoundation/ixo-cosmos/x/ixo"
)

const CURRENCY = "ixo-atom"

func NewHandler(k ProjectKeeper, ck bank.CoinKeeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		fmt.Println("Handler")
		fmt.Println(msg)
		switch msg := msg.(type) {
		case CreateProjectMsg:
			return handleCreateProjectMsg(ctx, k, ck, msg)
		case CreateAgentMsg:
			return handleCreateAgentMsg(ctx, k, ck, msg)
		case UpdateAgentMsg:
			return handleUpdateAgentMsg(ctx, k, ck, msg)
		case CreateClaimMsg:
			return handleCreateClaimMsg(ctx, k, ck, msg)
		case CreateEvaluationMsg:
			return handleCreateEvaluationMsg(ctx, k, ck, msg)
		case FundProjectMsg:
			return handleFundProjectMsg(ctx, k, ck, msg)
		case WithdrawFundsMsg:
			return handleWithdrawFundsMsg(ctx, k, ck, msg)
		default:
			return sdk.ErrUnknownRequest("No match for message type.").Result()
		}
	}
}

func handleCreateProjectMsg(ctx sdk.Context, k ProjectKeeper, ck bank.CoinKeeper, msg CreateProjectMsg) sdk.Result {
	fmt.Println("Handler")
	fmt.Println(msg)
	fmt.Println(msg.ProjectDoc)
	newProjectDoc := msg.ProjectDoc
	addAccountToAccountProjectAccounts(ctx, k, newProjectDoc.GetProjectDid(), newProjectDoc.GetProjectDid())

	fmt.Println(newProjectDoc)
	projectDoc, err := k.AddProjectDoc(ctx, newProjectDoc)
	if err != nil {
		return err.Result()
	}
	return sdk.Result{
		Code: sdk.CodeOK,
		Data: k.pm.encodeProject(projectDoc),
	}
}

func handleCreateAgentMsg(ctx sdk.Context, k ProjectKeeper, ck bank.CoinKeeper, msg CreateAgentMsg) sdk.Result {
	return sdk.Result{
		Code: sdk.CodeOK,
		Data: []byte("Action complete"),
	}
}
func handleUpdateAgentMsg(ctx sdk.Context, k ProjectKeeper, ck bank.CoinKeeper, msg UpdateAgentMsg) sdk.Result {
	return sdk.Result{
		Code: sdk.CodeOK,
		Data: []byte("Action complete"),
	}
}
func handleCreateClaimMsg(ctx sdk.Context, k ProjectKeeper, ck bank.CoinKeeper, msg CreateClaimMsg) sdk.Result {
	return sdk.Result{
		Code: sdk.CodeOK,
		Data: []byte("Action complete"),
	}
}
func handleCreateEvaluationMsg(ctx sdk.Context, k ProjectKeeper, ck bank.CoinKeeper, msg CreateEvaluationMsg) sdk.Result {
	evaluationDoc := msg.Data
	projectDoc, found := getProjectDoc(ctx, k, evaluationDoc.ProjectDid)
	if !found {
		return sdk.Result{
			Code: sdk.CodeInvalidAddress,
			Data: []byte("Could not find Project"),
		}
	}
	accMap := getProjectAccountMap(ctx, k, evaluationDoc.ProjectDid)
	projectAddrInterface, found := accMap[evaluationDoc.ProjectDid]
	if !found {
		return sdk.Result{
			Code: sdk.CodeInvalidAddress,
			Data: []byte("Could not find Project Account"),
		}
	}
	projectAddr := projectAddrInterface.(string)
	senderAccAddrInterface, found := accMap[evaluationDoc.SenderDid]
	var senderAccAddr string
	if !found {
		newAcc := addAccountToAccountProjectAccounts(ctx, k, evaluationDoc.ProjectDid, evaluationDoc.SenderDid)

		senderAccAddr = hex.EncodeToString(newAcc.GetAddress())
	} else {
		senderAccAddr = senderAccAddrInterface.(string)
	}
	err := ck.SendCoins(ctx, toHexBytes(projectAddr), toHexBytes(senderAccAddr), sdk.Coins{{COIN_DENOM, projectDoc.GetEvaluatorPay()}})
	if err != nil {
		panic(err)
	}
	return sdk.Result{
		Code: sdk.CodeOK,
		Data: []byte("Action complete"),
	}
}

func handleFundProjectMsg(ctx sdk.Context, k ProjectKeeper, ck bank.CoinKeeper, msg FundProjectMsg) sdk.Result {
	fundProjectDoc := msg.Data
	_, found := getProjectDoc(ctx, k, fundProjectDoc.ProjectDid)
	if !found {
		return sdk.Result{
			Code: sdk.CodeInvalidAddress,
			Data: []byte("Could not find Project"),
		}
	}
	accMap := getProjectAccountMap(ctx, k, fundProjectDoc.ProjectDid)
	projectAddrInterface, found := accMap[fundProjectDoc.ProjectDid]
	if !found {
		return sdk.Result{
			Code: sdk.CodeInvalidAddress,
			Data: []byte("Could not find Project Account"),
		}
	}
	projectAddr := projectAddrInterface.(string)

	_, err := ck.AddCoins(ctx, toHexBytes(projectAddr), sdk.Coins{{COIN_DENOM, fundProjectDoc.GetAmount()}})
	if err != nil {
		panic(err)
	}

	return sdk.Result{
		Code: sdk.CodeOK,
		Data: []byte("Action complete"),
	}
}
func handleWithdrawFundsMsg(ctx sdk.Context, k ProjectKeeper, ck bank.CoinKeeper, msg WithdrawFundsMsg) sdk.Result {
	return sdk.Result{
		Code: sdk.CodeOK,
		Data: []byte("Action complete"),
	}
}

func toHexBytes(address string) common.HexBytes {
	bz, err := hex.DecodeString(address)
	if err != nil {
		panic(err)
	}
	return sdk.Address(bz)
}

func getProjectDoc(ctx sdk.Context, k ProjectKeeper, projectDid ixo.Did) (ProjectDoc, bool) {
	ixoProjectDoc, found := k.GetProjectDoc(ctx, projectDid)
	return ixoProjectDoc.(ProjectDoc), found
}

func getProjectAccountMap(ctx sdk.Context, k ProjectKeeper, projectDid ixo.Did) map[string]interface{} {
	return k.GetAccountMap(ctx, projectDid)
}

func addAccountToAccountProjectAccounts(ctx sdk.Context, k ProjectKeeper, projectDid ixo.Did, accountDid ixo.Did) sdk.Account {
	acc := k.CreateNewAccount(ctx, projectDid, accountDid)
	k.AddAccountToAccountProjectAccounts(ctx, projectDid, accountDid, acc)

	return acc
}
