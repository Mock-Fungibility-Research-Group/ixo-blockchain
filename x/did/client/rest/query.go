package rest

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/ixofoundation/ixo-blockchain/x/did/exported"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/gorilla/mux"

	"github.com/ixofoundation/ixo-blockchain/x/did/internal/keeper"
	"github.com/ixofoundation/ixo-blockchain/x/did/internal/types"
)

func registerQueryRoutes(cliCtx context.CLIContext, r *mux.Router) {
	// The .* is necessary so that a slash in the did gets included as part of the did
	r.HandleFunc("/didToAddr/{did:.*}", queryAddressFromDidRequestHandler(cliCtx)).Methods("GET")
	r.HandleFunc("/pubKeyToAddr/{pubKey}", queryAddressFromBase58EncodedPubkeyRequestHandler(cliCtx)).Methods("GET")
	r.HandleFunc("/did/{did}", queryDidDocRequestHandler(cliCtx)).Methods("GET")
	r.HandleFunc("/did", queryAllDidsRequestHandler(cliCtx)).Methods("GET")
	r.HandleFunc("/allDidDocs", queryAllDidDocsRequestHandler(cliCtx)).Methods("GET")
}

func queryAddressFromBase58EncodedPubkeyRequestHandler(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)

		if !types.IsValidPubKey(vars["pubKey"]) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("input is not a valid base-58 encoded pubKey"))
			return
		}

		accAddress := exported.VerifyKeyToAddr(vars["pubKey"])

		rest.PostProcessResponse(w, cliCtx, accAddress)
	}
}

func queryAddressFromDidRequestHandler(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		didAddr := vars["did"]
		key := exported.Did(didAddr)
		res, _, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s/%s", types.QuerierRoute,
			keeper.QueryDidDoc, key), nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		if len(res) == 0 {
			w.WriteHeader(http.StatusNoContent)
			_, _ = w.Write([]byte("No data for respected did address."))
			return
		}

		var didDoc types.BaseDidDoc
		cliCtx.Codec.MustUnmarshalJSON(res, &didDoc)
		addressFromDid := didDoc.Address()

		rest.PostProcessResponse(w, cliCtx, addressFromDid)
	}
}

func queryDidDocRequestHandler(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		didAddr := vars["did"]
		key := exported.Did(didAddr)
		res, _, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s/%s", types.QuerierRoute,
			keeper.QueryDidDoc, key), nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(fmt.Sprintf("Could't query did. Error: %s", err.Error())))
			return
		}
		if len(res) == 0 {
			w.WriteHeader(http.StatusNoContent)
			_, _ = w.Write([]byte("No data for respected did address."))
			return
		}

		var didDoc types.BaseDidDoc
		cliCtx.Codec.MustUnmarshalJSON(res, &didDoc)

		rest.PostProcessResponse(w, cliCtx, didDoc)
	}
}

func queryAllDidsRequestHandler(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		res, _, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute,
			keeper.QueryAllDids), nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(fmt.Sprintf("Could't query did. Error: %s", err.Error())))
			return
		}

		if len(res) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var dids []exported.Did
		cliCtx.Codec.MustUnmarshalJSON(res, &dids)

		rest.PostProcessResponse(w, cliCtx, dids)
	}
}

func queryAllDidDocsRequestHandler(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		res, _, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute,
			keeper.QueryAllDidDocs), nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(fmt.Sprintf("Could't query did. Error: %s", err.Error())))
			return
		}

		if len(res) == 0 {
			w.WriteHeader(http.StatusNoContent)
			_, _ = w.Write([]byte("No data present."))
			return
		}

		var didDocs []types.BaseDidDoc
		cliCtx.Codec.MustUnmarshalJSON(res, &didDocs)

		rest.PostProcessResponse(w, cliCtx, didDocs)
	}
}
