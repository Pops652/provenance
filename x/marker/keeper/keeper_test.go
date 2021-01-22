package keeper_test

import (
	"testing"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	simapp "github.com/provenance-io/provenance/app"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/require"

	"github.com/provenance-io/provenance/x/marker/types"
)

func TestAccountMapperGetSet(t *testing.T) {
	//app, ctx := createTestApp(true)
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	addr := types.MustGetMarkerAddress("testcoin")
	user := testUserAddress("test")

	// no account before its created
	acc := app.AccountKeeper.GetAccount(ctx, addr)
	require.Nil(t, acc)

	// create account and check default values
	acc = types.NewEmptyMarkerAccount("testcoin", user.String(), nil)
	mac, ok := acc.(types.MarkerAccountI)
	require.True(t, ok)
	require.NotNil(t, mac)
	require.Equal(t, addr, mac.GetAddress())
	require.EqualValues(t, nil, mac.GetPubKey())

	// NewAccount doesn't call Set, so it's still nil
	require.Nil(t, app.AccountKeeper.GetAccount(ctx, addr))

	// set some values on the account and save it
	require.NoError(t, mac.GrantAccess(types.NewAccessGrant(user, []types.Access{types.Access_Mint, types.Access_Admin})))

	app.AccountKeeper.SetAccount(ctx, mac)

	// check the new values
	acc = app.AccountKeeper.GetAccount(ctx, addr)
	require.NotNil(t, acc)
	mac, ok = acc.(types.MarkerAccountI)
	require.True(t, ok)
	require.True(t, mac.AddressHasAccess(user, types.Access_Admin))

	app.MarkerKeeper.RemoveMarker(ctx, mac)

	// getting account after delete should be nil
	acc = app.AccountKeeper.GetAccount(ctx, addr)
	require.Nil(t, acc)

	// check for error on invaid marker denom
	_, err := app.MarkerKeeper.GetMarkerByDenom(ctx, "doesntexist")
	require.Error(t, err, "marker does not exist, should error")
}

func TestAccountKeeperReader(t *testing.T) {
	//app, ctx := createTestApp(true)
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	addr := types.MustGetMarkerAddress("testcoin")
	user := testUserAddress("test")
	// create account and check default values
	mac := types.NewEmptyMarkerAccount(
		"testcoin",
		user.String(),
		[]types.AccessGrant{*types.NewAccessGrant(user, []types.Access{types.Access_Mint})})

	require.NoError(t, app.MarkerKeeper.AddMarkerAccount(ctx, mac))

	m, err := app.MarkerKeeper.GetMarkerByDenom(ctx, "testcoin")
	require.NoError(t, err)
	require.NotNil(t, m)
	require.EqualValues(t, m.GetDenom(), "testcoin")
	require.EqualValues(t, m.GetAddress(), addr)

	m, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	require.NotNil(t, m)
	require.EqualValues(t, m.GetDenom(), "testcoin")
	require.EqualValues(t, m.GetAddress(), addr)

	count := 0
	app.MarkerKeeper.IterateMarkers(ctx, func(record types.MarkerAccountI) bool {
		require.EqualValues(t, record.GetDenom(), "testcoin")
		count++
		return false
	})
	require.EqualValues(t, count, 1)
}

// nolint:funlen
func TestAccountKeeperManageAccess(t *testing.T) {
	//app, ctx := createTestApp(true)
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	addr := types.MustGetMarkerAddress("testcoin")
	// Easiest way to create a valid bech32 address for testing.
	user1 := testUserAddress("test1")
	user2 := testUserAddress("test2")
	admin := testUserAddress("admin")

	// create account and check default values
	mac := types.NewEmptyMarkerAccount("testcoin",
		user1.String(),
		[]types.AccessGrant{*types.NewAccessGrant(user1, []types.Access{types.Access_Burn}),
			*types.NewAccessGrant(admin, []types.Access{types.Access_Admin})})

	require.NoError(t, mac.SetManager(user1))
	require.NoError(t, mac.SetSupply(sdk.NewCoin(mac.Denom, sdk.OneInt())))
	require.NoError(t, app.MarkerKeeper.AddMarkerAccount(ctx, mac))

	// Initial, should not have access
	m, err := app.MarkerKeeper.GetMarkerByDenom(ctx, "testcoin")
	require.NoError(t, err)
	require.NotNil(t, m)
	require.False(t, m.AddressHasAccess(user2, types.Access_Burn))

	// Grant access and check (succeeds on a Proposed marker without Admin grant)
	require.NoError(t,
		app.MarkerKeeper.AddAccess(
			ctx, user1, "testcoin", types.NewAccessGrant(user2, []types.Access{types.Access_Mint, types.Access_Delete})),
	)

	// Grant access fails for caller that is not the manager of a proposed marker
	require.Error(t, app.MarkerKeeper.AddAccess(
		ctx, user2, "testcoin", types.NewAccessGrant(user2, []types.Access{types.Access_Burn})))
	require.Error(t, app.MarkerKeeper.RemoveAccess(ctx, user2, "testcoin", user1))

	m, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	require.NotNil(t, m)
	require.True(t, m.AddressHasAccess(user2, types.Access_Mint))
	require.False(t, m.AddressHasAccess(user2, types.Access_Burn))
	require.False(t, m.AddressHasAccess(user2, types.Access_Admin))
	require.True(t, m.AddressHasAccess(user2, types.Access_Delete))
	require.False(t, m.AddressHasAccess(user2, types.Access_Withdraw))

	// Remove access and check
	require.NoError(t, app.MarkerKeeper.RemoveAccess(ctx, user1, "testcoin", user2))

	m, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	require.NotNil(t, m)
	require.False(t, m.AddressHasAccess(user2, types.Access_Mint))
	require.False(t, m.AddressHasAccess(user2, types.Access_Burn))
	require.False(t, m.AddressHasAccess(user2, types.Access_Admin))
	require.False(t, m.AddressHasAccess(user2, types.Access_Delete))
	require.False(t, m.AddressHasAccess(user2, types.Access_Withdraw))

	// Finalize marker and check permission enforcement.
	require.NoError(t, app.MarkerKeeper.FinalizeMarker(ctx, user1, m.GetDenom()))
	_, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)

	// Grant access and check (fails on a Finalized marker without Admin grant)
	require.Error(t, app.MarkerKeeper.AddAccess(ctx, user1, "testcoin",
		types.NewAccessGrant(user2, []types.Access{types.Access_Mint})))
	// Remove access fails for Finalized Marker without Admin grant
	require.Error(t, app.MarkerKeeper.RemoveAccess(ctx, user1, "testcoin", user2))

	// Admin can make changes to grants for active markers
	require.NoError(t, app.MarkerKeeper.AddAccess(ctx, admin, "testcoin",
		types.NewAccessGrant(user2, []types.Access{types.Access_Mint, types.Access_Delete})))
	_, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)

	// User2 can adjust supply up/down for a finalized marker
	require.NoError(t, app.MarkerKeeper.MintCoin(ctx, user2, sdk.NewCoin("testcoin", sdk.OneInt())))
	require.NoError(t, app.MarkerKeeper.BurnCoin(ctx, user1, sdk.NewCoin("testcoin", sdk.OneInt())))

	// Cancel marker and check permission enforcement.
	require.NoError(t, app.MarkerKeeper.CancelMarker(ctx, user2, "testcoin"))

	// Admin cannot make changes to grants for cancelled markers
	require.Error(t, app.MarkerKeeper.AddAccess(ctx, admin, "testcoin",
		types.NewAccessGrant(user2, []types.Access{types.Access_Burn})))
	require.Error(t, app.MarkerKeeper.RemoveAccess(ctx, admin, "testcoin", user2))

	// Load the marker one last time and verify our permission records are consistent and correct
	m, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)

	require.True(t, m.AddressHasAccess(admin, types.Access_Admin))
	require.True(t, m.AddressHasAccess(user1, types.Access_Burn))
	require.True(t, m.AddressHasAccess(user2, types.Access_Mint))
	require.True(t, m.AddressHasAccess(user2, types.Access_Delete))

	require.EqualValues(t, 1, len(m.AddressListForPermission(types.Access_Delete)))
	require.EqualValues(t, 1, len(m.AddressListForPermission(types.Access_Burn)))
	require.EqualValues(t, 1, len(m.AddressListForPermission(types.Access_Admin)))
	require.EqualValues(t, 0, len(m.AddressListForPermission(types.Access_Deposit)))
	require.EqualValues(t, 1, len(m.AddressListForPermission(types.Access_Mint)))
	require.EqualValues(t, 0, len(m.AddressListForPermission(types.Access_Withdraw)))
}

func TestAccountKeeperCancelProposedByManager(t *testing.T) {
	//app, ctx := createTestApp(true)
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	addr := types.MustGetMarkerAddress("testcoin")
	// Easiest way to create a valid bech32 address for testing.
	user1 := testUserAddress("test1")
	user2 := testUserAddress("test2")
	admin := testUserAddress("admin")

	// create account and check default values
	mac := types.NewEmptyMarkerAccount("testcoin",
		user1.String(),
		[]types.AccessGrant{*types.NewAccessGrant(user1, []types.Access{types.Access_Burn}),
			*types.NewAccessGrant(admin, []types.Access{types.Access_Admin})})

	require.NoError(t, mac.SetManager(user1))
	require.NoError(t, mac.SetSupply(sdk.NewCoin(mac.Denom, sdk.OneInt())))
	require.NoError(t, app.MarkerKeeper.AddMarkerAccount(ctx, mac))

	m, err := app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	// user1 and user2 will not have been assigned delete
	require.False(t, m.AddressHasAccess(user1, types.Access_Delete))
	require.False(t, m.AddressHasAccess(user2, types.Access_Delete))

	// Cancel marker and check permission enforcement. (expect fail, no access)
	require.Error(t, app.MarkerKeeper.CancelMarker(ctx, user2, "testcoin"))

	// Cancel marker and check permission enforcement. (succeeds for manager)
	require.NoError(t, app.MarkerKeeper.CancelMarker(ctx, user1, "testcoin"))
}

// nolint:funlen
func TestAccountKeeperMintBurnCoins(t *testing.T) {
	//app, ctx := createTestApp(true)
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	addr := types.MustGetMarkerAddress("testcoin")
	user := testUserAddress("test")

	// fail for an unknown coin.
	require.Error(t, app.MarkerKeeper.MintCoin(ctx, user, sdk.NewInt64Coin("testcoin", 100)))
	require.Error(t, app.MarkerKeeper.BurnCoin(ctx, user, sdk.NewInt64Coin("testcoin", 100)))

	// create account and check default values
	mac := types.NewEmptyMarkerAccount("testcoin", user.String(), []types.AccessGrant{*types.NewAccessGrant(user,
		[]types.Access{types.Access_Mint, types.Access_Burn, types.Access_Withdraw, types.Access_Delete})})
	require.NoError(t, mac.SetManager(user))
	require.NoError(t, mac.SetSupply(sdk.NewCoin("testcoin", sdk.NewInt(1000))))

	require.NoError(t, app.MarkerKeeper.AddMarkerAccount(ctx, mac))
	// Should not fail for a non-active/finalized coin, must be able to adjust supply amount to match any existing
	require.NoError(t, app.MarkerKeeper.MintCoin(ctx, user, sdk.NewInt64Coin("testcoin", 100)))
	require.NoError(t, app.MarkerKeeper.BurnCoin(ctx, user, sdk.NewInt64Coin("testcoin", 100)))

	// Moves to finalized, mints required supply, moves to active status.
	require.NoError(t, app.MarkerKeeper.FinalizeMarker(ctx, user, "testcoin"))
	require.NoError(t, app.MarkerKeeper.ActivateMarker(ctx, user, "testcoin"))

	// Load the created marker
	m, err := app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	require.EqualValues(t, m.GetSupply(), sdk.NewInt64Coin("testcoin", 1000))
	// entire supply should have been allocated to markeracount
	require.EqualValues(t, app.MarkerKeeper.GetEscrow(ctx, m).AmountOf("testcoin"), sdk.NewInt(1000))

	// perform a successful mint (and check)
	require.NoError(t, app.MarkerKeeper.MintCoin(ctx, user, sdk.NewInt64Coin("testcoin", 100)))
	m, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	require.EqualValues(t, m.GetSupply(), sdk.NewInt64Coin("testcoin", 1100))
	require.EqualValues(t, app.MarkerKeeper.GetEscrow(ctx, m), sdk.NewCoins(sdk.NewInt64Coin("testcoin", 1100)))

	// perform a successful burn (and check)
	require.NoError(t, app.MarkerKeeper.BurnCoin(ctx, user, sdk.NewInt64Coin("testcoin", 100)))
	m, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	require.EqualValues(t, m.GetSupply(), sdk.NewInt64Coin("testcoin", 1000))

	// Fail for burn too much (exceed supply)
	require.Error(t, app.MarkerKeeper.BurnCoin(ctx, user, sdk.NewInt64Coin("testcoin", 10000)))

	// check that supply remains unchanged after above error
	m, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	require.EqualValues(t, m.GetSupply(), sdk.NewInt64Coin("testcoin", 1000))

	// Check that our marker account is currently holding all the minted coins after above mint/burn
	require.EqualValues(t, app.MarkerKeeper.GetEscrow(ctx, m), sdk.NewCoins(sdk.NewInt64Coin("testcoin", 1000)))

	// move coin out of the marker and into a user account
	require.NoError(t, app.MarkerKeeper.WithdrawCoins(ctx, user, user, "testcoin",
		sdk.NewCoins(sdk.NewInt64Coin("testcoin", 50))))

	// verify user has the withdrawn coins
	require.EqualValues(t, app.BankKeeper.GetBalance(ctx, user, "testcoin").Amount, sdk.NewInt(50))

	// verify marker account has remaining coins
	m, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	require.EqualValues(t, app.MarkerKeeper.GetEscrow(ctx, m).AmountOf("testcoin"), sdk.NewInt(950))

	// Fail for burn too much (exceed marker account holdings)
	require.Error(t, app.MarkerKeeper.BurnCoin(ctx, user, sdk.NewInt64Coin("testcoin", 1000)))
	// Fails because a user is holding some of the supply
	require.Error(t, app.MarkerKeeper.CancelMarker(ctx, user, "testcoin"))

	// put the coins back in the types.
	require.NoError(t, app.BankKeeper.SendCoins(ctx, user, addr, sdk.NewCoins(sdk.NewInt64Coin("testcoin", 50))))
	// succeeds because marker has all its supply
	require.NoError(t, app.MarkerKeeper.CancelMarker(ctx, user, "testcoin"))

	// verify status is cancelled
	m, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	require.EqualValues(t, types.StatusCancelled, m.GetStatus())

	// succeeds on a cancelled marker (no-op)
	require.NoError(t, app.MarkerKeeper.CancelMarker(ctx, user, "testcoin"))

	require.NoError(t, app.MarkerKeeper.DeleteMarker(ctx, user, "testcoin"))

	// verify status is destroyed and supply is zero.
	m, err = app.MarkerKeeper.GetMarker(ctx, addr)
	require.NoError(t, err)
	require.EqualValues(t, types.StatusDestroyed, m.GetStatus())
	require.EqualValues(t, m.GetSupply().Amount, sdk.ZeroInt())

	// supply module should also indicate a zero supply
	require.EqualValues(t, app.BankKeeper.GetSupply(ctx).GetTotal().AmountOf("testcoin"), sdk.ZeroInt())
}

func TestAccountKeeperGetAll(t *testing.T) {
	//app, ctx := createTestApp(true)
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	user := testUserAddress("test")
	mac := types.NewEmptyMarkerAccount("testcoin",
		user.String(),
		[]types.AccessGrant{*types.NewAccessGrant(user, []types.Access{types.Access_Deposit})})
	require.NoError(t, mac.SetManager(user))
	require.NoError(t, app.MarkerKeeper.AddMarkerAccount(ctx, mac))

	mac = types.NewEmptyMarkerAccount("secondcoin",
		user.String(),
		[]types.AccessGrant{*types.NewAccessGrant(user, []types.Access{types.Access_Deposit})})
	require.NoError(t, mac.SetManager(user))
	require.NoError(t, app.MarkerKeeper.AddMarkerAccount(ctx, mac))

	var err error
	var m types.MarkerAccountI
	m, err = app.MarkerKeeper.GetMarkerByDenom(ctx, "testcoin")
	require.NoError(t, err)
	require.NotNil(t, m)

	m, err = app.MarkerKeeper.GetMarkerByDenom(ctx, "secondcoin")
	require.NoError(t, err)
	require.NotNil(t, m)

	count := 0
	app.MarkerKeeper.IterateMarkers(ctx, func(record types.MarkerAccountI) bool {
		count++
		return false
	})
	require.Equal(t, 2, count)

	// Could do more in-depth checking, but if both markers are returned that is the expected behavior
}

func TestAccountInsufficientExisting(t *testing.T) {
	//app, ctx := createTestApp(true)
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	pubkey := secp256k1.GenPrivKey().PubKey()
	user := sdk.AccAddress(pubkey.Address())

	// setup an existing account with an existing balance (and matching supply)
	existingSupply := sdk.NewCoin("testcoin", sdk.NewInt(10000))
	app.AccountKeeper.SetAccount(ctx, authtypes.NewBaseAccount(user, pubkey, 0, 0))
	app.BankKeeper.SetBalance(ctx, user, existingSupply)
	prevSupply := app.BankKeeper.GetSupply(ctx)
	app.BankKeeper.SetSupply(ctx, banktypes.NewSupply(prevSupply.GetTotal().Add(existingSupply)))

	// create account and check default values
	mac := types.NewEmptyMarkerAccount("testcoin", user.String(), []types.AccessGrant{*types.NewAccessGrant(user,
		[]types.Access{types.Access_Mint, types.Access_Burn, types.Access_Withdraw, types.Access_Delete})})
	require.NoError(t, mac.SetManager(user))
	require.NoError(t, mac.SetSupply(sdk.NewCoin("testcoin", sdk.NewInt(1000))))

	require.NoError(t, app.MarkerKeeper.AddMarkerAccount(ctx, mac))
	// insufficient supply to cover existing
	require.Error(t, app.MarkerKeeper.FinalizeMarker(ctx, user, "testcoin"))

	// move supply up high enough
	require.NoError(t, app.MarkerKeeper.MintCoin(ctx, user, sdk.NewInt64Coin("testcoin", 9001)))
	// no error now...
	require.NoError(t, app.MarkerKeeper.FinalizeMarker(ctx, user, "testcoin"))
	require.NoError(t, app.MarkerKeeper.ActivateMarker(ctx, user, "testcoin"))

	var err error
	var m types.MarkerAccountI
	m, err = app.MarkerKeeper.GetMarkerByDenom(ctx, "testcoin")
	require.NoError(t, err)
	require.NotNil(t, m)

	// Amount that was minted shal be 1
	require.EqualValues(t, 1, app.MarkerKeeper.GetEscrow(ctx, m).AmountOf("testcoin").Int64())
	// Amount of the total supply shall be 10001
	require.EqualValues(t, 10001, m.GetSupply().Amount.Int64())
}

func TestAccountImplictControl(t *testing.T) {
	//app, ctx := createTestApp(true)
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	user := testUserAddress("test")
	user2 := testUserAddress("test2")

	// create account and check default values
	mac := types.NewEmptyMarkerAccount("testcoin", user.String(), []types.AccessGrant{*types.NewAccessGrant(user,
		[]types.Access{types.Access_Mint, types.Access_Burn, types.Access_Withdraw, types.Access_Delete})})

	mac.MarkerType = types.MarkerType_RestrictedCoin
	require.NoError(t, mac.SetManager(user))
	require.NoError(t, mac.SetSupply(sdk.NewCoin("testcoin", sdk.NewInt(1000))))

	require.NoError(t, app.MarkerKeeper.AddMarkerAccount(ctx, mac))

	// Moves to finalized, mints required supply, moves to active status.
	require.NoError(t, app.MarkerKeeper.FinalizeMarker(ctx, user, "testcoin"))
	// No send enabled flag enforced yet, default is allowed
	require.True(t, app.BankKeeper.SendEnabledCoin(ctx, sdk.NewCoin("testcoin", sdk.NewInt(10))))
	require.NoError(t, app.MarkerKeeper.ActivateMarker(ctx, user, "testcoin"))
	// Activated restricted coins can not be sent directly, verify is false now
	require.False(t, app.BankKeeper.SendEnabledCoin(ctx, sdk.NewCoin("testcoin", sdk.NewInt(10))))

	// Must fail because user2 does not have any access
	require.Error(t, app.MarkerKeeper.AddAccess(ctx, user2, "testcoin", types.NewAccessGrant(user2,
		[]types.Access{types.Access_Mint, types.Access_Delete})))

	// Move 100% of the supply into user2.
	require.NoError(t, app.MarkerKeeper.WithdrawCoins(ctx, user, user2, "testcoin",
		sdk.NewCoins(sdk.NewInt64Coin("testcoin", 1000))))

	// Succeeds now because user2 is holding all of the testcoin supply.
	require.NoError(t, app.MarkerKeeper.AddAccess(ctx, user2, "testcoin",
		types.NewAccessGrant(user2, []types.Access{types.Access_Mint, types.Access_Delete, types.Access_Transfer})))

	// succeeds for a user with transfer rights
	require.NoError(t, app.MarkerKeeper.TransferCoin(ctx, user2, user, user2, sdk.NewCoin("testcoin", sdk.NewInt(10))))
	// fails if the admin user does not have transfer authority
	require.Error(t, app.MarkerKeeper.TransferCoin(ctx, user, user2, user, sdk.NewCoin("testcoin", sdk.NewInt(10))))
}

// testUserAddress gives a quick way to make a valid test address (no keys though)
func testUserAddress(name string) sdk.AccAddress {
	addr := types.MustGetMarkerAddress(name)
	return addr
}
