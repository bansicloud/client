package keybase_1

import (
	"net/rpc"
)

type GenericClient interface {
	Call(s string, args interface{}, res interface{}) error
}

type Status struct {
	Code   int      `codec:"code"`
	Name   string   `codec:"name"`
	Desc   string   `codec:"desc"`
	Fields []string `codec:"fields"`
}

type UID [16]byte
type GetCurrentStatusArg struct {
}

type GetCurrentStatusResBody struct {
	Configured        bool `codec:"configured"`
	Registered        bool `codec:"registered"`
	LoggedIn          bool `codec:"loggedIn"`
	PublicKeySelected bool `codec:"publicKeySelected"`
	HasPrivateKey     bool `codec:"hasPrivateKey"`
}

type GetCurrentStatusRes struct {
	Body   *GetCurrentStatusResBody `codec:"body,omitempty"`
	Status Status                   `codec:"status"`
}

type ConfigInterface interface {
	GetCurrentStatus(arg *GetCurrentStatusArg, res *GetCurrentStatusRes) error
}

func RegisterConfig(server *rpc.Server, i ConfigInterface) error {
	return server.RegisterName("keybase.1.config", i)
}

type ConfigClient struct {
	cli GenericClient
}

func (c ConfigClient) GetCurrentStatus(arg GetCurrentStatusArg, res *GetCurrentStatusRes) error {
	return c.Call("keybase.1.config.GetCurrentStatus", arg, res)
}

type LoginResBody struct {
	Uid UID `codec:"uid"`
}

type IsLoggedInRes struct {
	Body   *LoginResBody `codec:"body,omitempty"`
	Status Status        `codec:"status"`
}

type IsLoggedInArg struct {
}

type PasswordLoginArg struct {
	Password string `codec:"password"`
}

type PubkeyLoginArg struct {
}

type PasswordLoginRes struct {
	Body   *LoginResBody `codec:"body,omitempty"`
	Status Status        `codec:"status"`
}

type PubkeyLoginRes struct {
	Body   *LoginResBody `codec:"body,omitempty"`
	Status Status        `codec:"status"`
}

type LogoutRes struct {
	Status Status `codec:"status"`
}

type LogoutArg struct {
}

type SwitchUserRes struct {
	Status Status `codec:"status"`
}

type SwitchUserArg struct {
	Username string `codec:"username"`
}

type LoginInterface interface {
	IsLoggedIn(arg *IsLoggedInArg, res *IsLoggedInRes) error
	PasswordLogin(arg *PasswordLoginArg, res *PasswordLoginRes) error
	PubkeyLogin(arg *PubkeyLoginArg, res *PubkeyLoginRes) error
	Logout(arg *LogoutArg, res *LogoutRes) error
	SwitchUser(arg *SwitchUserArg, res *SwitchUserRes) error
}

func RegisterLogin(server *rpc.Server, i LoginInterface) error {
	return server.RegisterName("keybase.1.login", i)
}

type LoginClient struct {
	cli GenericClient
}

func (c LoginClient) IsLoggedIn(arg IsLoggedInArg, res *IsLoggedInRes) error {
	return c.Call("keybase.1.login.isLoggedIn", arg, res)
}

func (c LoginClient) PasswordLogin(arg PasswordLoginArg, res *PasswordLoginRes) error {
	return c.Call("keybase.1.login.passwordLogin", arg, res)
}

func (c LoginClient) PubkeyLogin(arg PubkeyLoginArg, res *PubkeyLoginRes) error {
	return c.Call("keybase.1.login.pubkeyLogin", arg, res)
}

func (c LoginClient) Logout(arg LogoutArg, res *LogoutRes) error {
	return c.Call("keybase.1.login.logout", arg, res)
}

func (c LoginClient) SwitchUser(arg SwitchUserArg, res *SwitchUserRes) error {
	return c.Call("keybase.1.login.switchUser", arg, res)
}

type CheckUsernameAvailableRes struct {
	Status Status `codec:"status"`
}

type CheckUsernameAvailableArg struct {
	Username string `codec:"username"`
}

type SignupArg struct {
	Email      string `codec:"email"`
	InviteCode string `codec:"inviteCode"`
	Passphrase string `codec:"passphrase"`
	Username   string `codec:"username"`
}

type InviteRequestArg struct {
	Email    string `codec:"email"`
	Fullname string `codec:"fullname"`
	Notes    string `codec:"notes"`
}

type InviteRequestResBody struct {
	Code  string `codec:"code"`
	Place int    `codec:"place"`
}

type InviteRequestRes struct {
	Body   *InviteRequestResBody `codec:"body,omitempty"`
	Status Status                `codec:"status"`
}

type SignupResSuccess struct {
	Uid UID `codec:"uid"`
}

type SignupResBody struct {
	Success      *SignupResSuccess `codec:"success,omitempty"`
	PassphraseOk bool              `codec:"passphraseOk"`
	PostOk       bool              `codec:"postOk"`
	WriteOk      bool              `codec:"writeOk"`
}

type SignupRes struct {
	Body   SignupResBody `codec:"body"`
	Status Status        `codec:"status"`
}

type SignupInterface interface {
	CheckUsernameAvailable(arg *CheckUsernameAvailableArg, res *CheckUsernameAvailableRes) error
	Signup(arg *SignupArg, res *SignupRes) error
	InviteResuest(arg *InviteRequestArg, res *InviteRequestRes) error
}

func RegisterSignup(server *rpc.Server, i SignupInterface) error {
	return server.RegisterName("keybase.1.signup", i)
}

type SignupClient struct {
	cli GenericClient
}

func (c SignupClient) CheckUsernameAvailable(arg CheckUsernameAvailableArg, res *CheckUsernameAvailableRes) error {
	return c.Call("keybase.1.signup.checkUsernameAvailable", arg, res)
}

func (c SignupClient) Signup(arg SignupArg, res *SignupRes) error {
	return c.Call("keybase.1.signup.signup", arg, res)
}

func (c SignupClient) InviteResuest(arg InviteRequestArg, res *InviteRequestRes) error {
	return c.Call("keybase.1.signup.inviteResuest", arg, res)
}
