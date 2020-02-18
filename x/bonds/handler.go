package bonds

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ixofoundation/ixo-cosmos/x/bonds/internal/keeper"
	"github.com/ixofoundation/ixo-cosmos/x/bonds/internal/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"strings"
)

func NewHandler(keeper keeper.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case types.MsgCreateBond:
			return handleMsgCreateBond(ctx, keeper, msg)
		case types.MsgEditBond:
			return handleMsgEditBond(ctx, keeper, msg)
		case types.MsgBuy:
			return handleMsgBuy(ctx, keeper, msg)
		case types.MsgSell:
			return handleMsgSell(ctx, keeper, msg)
		case types.MsgSwap:
			return handleMsgSwap(ctx, keeper, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized bonds Msg type: %v", msg.Type())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func EndBlocker(ctx sdk.Context, keeper keeper.Keeper) []abci.ValidatorUpdate {

	iterator := keeper.GetBondIterator(ctx)
	for ; iterator.Valid(); iterator.Next() {
		bond := keeper.MustGetBondByKey(ctx, iterator.Key())
		batch := keeper.MustGetBatch(ctx, bond.Token)

		// Subtract one block
		batch.BlocksRemaining = batch.BlocksRemaining.SubUint64(1)
		keeper.SetBatch(ctx, bond.Token, batch)

		// If blocks remaining > 0 do not perform orders
		if !batch.BlocksRemaining.IsZero() {
			continue
		}

		// Perform orders
		keeper.PerformOrders(ctx, bond.Token)

		// Get batch again just in case orders were cancelled
		batch = keeper.MustGetBatch(ctx, bond.Token)

		// Save current as last and reset current
		keeper.SetLastBatch(ctx, bond.Token, batch)
		keeper.SetBatch(ctx, bond.Token, types.NewBatch(bond.Token, bond.BatchBlocks))
	}
	return []abci.ValidatorUpdate{}
}

func handleMsgCreateBond(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgCreateBond) sdk.Result {

	if keeper.BondExists(ctx, msg.Token) {
		return types.ErrBondAlreadyExists(DefaultCodeSpace, msg.Token).Result()
	} else if msg.Token == keeper.StakingKeeper.GetParams(ctx).BondDenom {
		return types.ErrBondTokenCannotBeStakingToken(DefaultCodeSpace).Result()
	}

	reserveAddress := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

	bond := NewBond(msg.Token, msg.Name, msg.Description, msg.Creator,
		msg.FunctionType, msg.FunctionParameters, msg.ReserveTokens,
		reserveAddress, msg.TxFeePercentage, msg.ExitFeePercentage,
		msg.FeeAddress, msg.MaxSupply, msg.OrderQuantityLimits, msg.SanityRate,
		msg.SanityMarginPercentage, msg.AllowSells, msg.Signers, msg.BatchBlocks)

	keeper.SetBond(ctx, msg.Token, bond)
	keeper.SetBatch(ctx, msg.Token, types.NewBatch(bond.Token, msg.BatchBlocks))

	logger := keeper.Logger(ctx)
	logger.Info(fmt.Sprintf("bond %s with reserve(s) [%s] created by %s",
		msg.Token, strings.Join(bond.ReserveTokens, ","), msg.Creator.String()))

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateBond,
			sdk.NewAttribute(types.AttributeKeyBond, msg.Token),
			sdk.NewAttribute(types.AttributeKeyName, msg.Name),
			sdk.NewAttribute(types.AttributeKeyDescription, msg.Description),
			sdk.NewAttribute(types.AttributeKeyFunctionType, msg.FunctionType),
			sdk.NewAttribute(types.AttributeKeyFunctionParameters, msg.FunctionParameters.String()),
			sdk.NewAttribute(types.AttributeKeyReserveTokens, types.StringsToString(msg.ReserveTokens)),
			sdk.NewAttribute(types.AttributeKeyReserveAddress, reserveAddress.String()),
			sdk.NewAttribute(types.AttributeKeyTxFeePercentage, msg.TxFeePercentage.String()),
			sdk.NewAttribute(types.AttributeKeyExitFeePercentage, msg.ExitFeePercentage.String()),
			sdk.NewAttribute(types.AttributeKeyFeeAddress, msg.FeeAddress.String()),
			sdk.NewAttribute(types.AttributeKeyMaxSupply, msg.MaxSupply.String()),
			sdk.NewAttribute(types.AttributeKeyOrderQuantityLimits, msg.OrderQuantityLimits.String()),
			sdk.NewAttribute(types.AttributeKeySanityRate, msg.SanityRate.String()),
			sdk.NewAttribute(types.AttributeKeySanityMarginPercentage, msg.SanityMarginPercentage.String()),
			sdk.NewAttribute(types.AttributeKeyAllowSells, msg.AllowSells),
			sdk.NewAttribute(types.AttributeKeySigners, types.AccAddressesToString(msg.Signers)),
			sdk.NewAttribute(types.AttributeKeyBatchBlocks, msg.BatchBlocks.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Creator.String()),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgEditBond(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgEditBond) sdk.Result {

	bond, found := keeper.GetBond(ctx, msg.Token)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, msg.Token).Result()
	}

	if !bond.SignersEqualTo(msg.Signers) {
		errMsg := fmt.Sprintf("List of signers does not match the one in the bond")
		return sdk.ErrInternal(errMsg).Result()
	}

	if msg.Name != types.DoNotModifyField {
		bond.Name = msg.Name
	}
	if msg.Description != types.DoNotModifyField {
		bond.Description = msg.Description
	}

	if msg.OrderQuantityLimits != types.DoNotModifyField {
		orderQuantityLimits, err := sdk.ParseCoins(msg.OrderQuantityLimits)
		if err != nil {
			return sdk.ErrInternal(err.Error()).Result()
		}
		bond.OrderQuantityLimits = orderQuantityLimits
	}

	if msg.SanityRate != types.DoNotModifyField {
		var sanityRate, sanityMarginPercentage sdk.Dec
		if msg.SanityRate == "" {
			sanityRate = sdk.ZeroDec()
			sanityMarginPercentage = sdk.ZeroDec()
		} else {
			parsedSanityRate, err := sdk.NewDecFromStr(msg.SanityRate)
			if err != nil {
				return types.ErrArgumentMissingOrNonFloat(types.DefaultCodespace, "sanity rate").Result()
			} else if parsedSanityRate.IsNegative() {
				return types.ErrArgumentCannotBeNegative(types.DefaultCodespace, "sanity rate").Result()
			}
			parsedSanityMarginPercentage, err := sdk.NewDecFromStr(msg.SanityMarginPercentage)
			if err != nil {
				return types.ErrArgumentMissingOrNonFloat(types.DefaultCodespace, "sanity margin percentage").Result()
			} else if parsedSanityMarginPercentage.IsNegative() {
				return types.ErrArgumentCannotBeNegative(types.DefaultCodespace, "sanity margin percentage").Result()
			}
			sanityRate = parsedSanityRate
			sanityMarginPercentage = parsedSanityMarginPercentage
		}
		bond.SanityRate = sanityRate
		bond.SanityMarginPercentage = sanityMarginPercentage
	}

	logger := keeper.Logger(ctx)
	logger.Info(fmt.Sprintf("bond %s edited by %s",
		msg.Token, msg.Editor.String()))

	keeper.SetBond(ctx, msg.Token, bond)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeEditBond,
			sdk.NewAttribute(types.AttributeKeyBond, msg.Token),
			sdk.NewAttribute(types.AttributeKeyName, msg.Name),
			sdk.NewAttribute(types.AttributeKeyDescription, msg.Description),
			sdk.NewAttribute(types.AttributeKeyOrderQuantityLimits, msg.OrderQuantityLimits),
			sdk.NewAttribute(types.AttributeKeySanityRate, msg.SanityRate),
			sdk.NewAttribute(types.AttributeKeySanityMarginPercentage, msg.SanityMarginPercentage),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Editor.String()),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgBuy(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgBuy) sdk.Result {

	token := msg.Amount.Denom
	bond, found := keeper.GetBond(ctx, token)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, token).Result()
	}

	// Check max prices
	if !bond.ReserveDenomsEqualTo(msg.MaxPrices) {
		return types.ErrReserveDenomsMismatch(types.DefaultCodespace, msg.MaxPrices, bond.ReserveTokens).Result()
	}

	// Check if order quantity limit exceeded
	if bond.AnyOrderQuantityLimitsExceeded(sdk.Coins{msg.Amount}) {
		return types.ErrOrderQuantityLimitExceeded(types.DefaultCodespace).Result()
	}

	// For the swapper, the first buy is the initialisation of the reserves
	// The max prices are used as the actual prices and one token is minted
	// The amount of token serves to define the price of adding more liquidity
	if bond.CurrentSupply.IsZero() && bond.FunctionType == types.SwapperFunction {
		return performFirstSwapperFunctionBuy(ctx, keeper, msg)
	}

	// Take max that buyer is willing to pay (enforces maxPrice <= balance)
	err := keeper.SupplyKeeper.SendCoinsFromAccountToModule(ctx, msg.Buyer,
		types.BatchesIntermediaryAccount, msg.MaxPrices)
	if err != nil {
		return err.Result()
	}

	// Create order
	order := types.NewBuyOrder(msg.Buyer, msg.Amount, msg.MaxPrices)

	// Get buy price and check if can add buy order to batch
	buyPrices, sellPrices, err := keeper.GetUpdatedBatchPricesAfterBuy(ctx, token, order)
	if err != nil {
		return err.Result()
	}

	// Add buy order to batch
	keeper.AddBuyOrder(ctx, token, order, buyPrices, sellPrices)

	// Cancel unfulfillable orders
	keeper.CancelUnfulfillableOrders(ctx, token)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeBuy,
			sdk.NewAttribute(types.AttributeKeyBond, msg.Amount.Denom),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyMaxPrices, msg.MaxPrices.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Buyer.String()),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func performFirstSwapperFunctionBuy(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgBuy) sdk.Result {

	// TODO: investigate effect that a high amount has on future buyers' ability to buy.

	token := msg.Amount.Denom
	bond, found := keeper.GetBond(ctx, token)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, token).Result()
	}

	// Check if initial liquidity violates sanity rate
	if bond.ReservesViolateSanityRate(msg.MaxPrices) {
		return types.ErrValuesViolateSanityRate(types.DefaultCodespace).Result()
	}

	// Use max prices as the amount to send to the liquidity pool (i.e. price)
	err := keeper.CoinKeeper.SendCoins(ctx, msg.Buyer, bond.ReserveAddress, msg.MaxPrices)
	if err != nil {
		return err.Result()
	}

	// Mint bond tokens
	err = keeper.SupplyKeeper.MintCoins(ctx, types.BondsMintBurnAccount,
		sdk.Coins{msg.Amount})
	if err != nil {
		return err.Result()
	}

	// Send bond tokens to buyer
	err = keeper.SupplyKeeper.SendCoinsFromModuleToAccount(ctx,
		types.BondsMintBurnAccount, msg.Buyer, sdk.Coins{msg.Amount})
	if err != nil {
		return err.Result()
	}

	// Update supply
	keeper.SetCurrentSupply(ctx, bond.Token, bond.CurrentSupply.Add(msg.Amount))

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeInitSwapper,
			sdk.NewAttribute(types.AttributeKeyBond, msg.Amount.Denom),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyChargedPrices, msg.MaxPrices.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Buyer.String()),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgSell(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgSell) sdk.Result {

	token := msg.Amount.Denom
	bond, found := keeper.GetBond(ctx, token)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, token).Result()
	}

	if strings.ToLower(bond.AllowSells) == types.FALSE {
		return types.ErrBondDoesNotAllowSelling(types.DefaultCodespace).Result()
	}

	// Check if order quantity limit exceeded
	if bond.AnyOrderQuantityLimitsExceeded(sdk.Coins{msg.Amount}) {
		return types.ErrOrderQuantityLimitExceeded(types.DefaultCodespace).Result()
	}

	// Send coins to be burned from seller (enforces sellAmount <= balance)
	err := keeper.SupplyKeeper.SendCoinsFromAccountToModule(ctx, msg.Seller,
		types.BondsMintBurnAccount, sdk.Coins{msg.Amount})
	if err != nil {
		return err.Result()
	}

	// Burn bond tokens to be sold
	err = keeper.SupplyKeeper.BurnCoins(ctx, types.BondsMintBurnAccount,
		sdk.Coins{msg.Amount})
	if err != nil {
		return err.Result()
	}

	// Create order
	order := types.NewSellOrder(msg.Seller, msg.Amount)

	// Get sell price and check if can add sell order to batch
	buyPrices, sellPrices, err := keeper.GetUpdatedBatchPricesAfterSell(ctx, token, order)
	if err != nil {
		return err.Result()
	}

	// Add sell order to batch
	keeper.AddSellOrder(ctx, token, order, buyPrices, sellPrices)

	//// Cancel unfulfillable orders (Note: no need)
	//keeper.CancelUnfulfillableOrders(ctx, token)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSell,
			sdk.NewAttribute(types.AttributeKeyBond, msg.Amount.Denom),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.Amount.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Seller.String()),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgSwap(ctx sdk.Context, keeper keeper.Keeper, msg types.MsgSwap) sdk.Result {

	bond, found := keeper.GetBond(ctx, msg.BondToken)
	if !found {
		return types.ErrBondDoesNotExist(types.DefaultCodespace, msg.BondToken).Result()
	}

	// Check if order quantity limit exceeded
	if bond.AnyOrderQuantityLimitsExceeded(sdk.Coins{msg.From}) {
		return types.ErrOrderQuantityLimitExceeded(types.DefaultCodespace).Result()
	}

	// Take coins to be swapped from swapper (enforces swapAmount <= balance)
	err := keeper.SupplyKeeper.SendCoinsFromAccountToModule(ctx, msg.Swapper,
		types.BatchesIntermediaryAccount, sdk.Coins{msg.From})
	if err != nil {
		return err.Result()
	}

	// Create order
	order := types.NewSwapOrder(msg.Swapper, msg.From, msg.ToToken)

	// Add swap order to batch
	keeper.AddSwapOrder(ctx, msg.BondToken, order)

	//// Cancel unfulfillable orders (Note: no need)
	//keeper.CancelUnfulfillableOrders(ctx, token)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSwap,
			sdk.NewAttribute(types.AttributeKeyBond, msg.BondToken),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.From.Amount.String()),
			sdk.NewAttribute(types.AttributeKeySwapFromToken, msg.From.Denom),
			sdk.NewAttribute(types.AttributeKeySwapToToken, msg.ToToken),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Swapper.String()),
		),
	})

	return sdk.Result{Events: ctx.EventManager().Events()}
}
