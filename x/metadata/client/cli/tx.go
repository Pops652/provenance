package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/provenance-io/provenance/x/metadata/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	FlagSigners = "signers"
)

// NewTxCmd is the top-level command for Metadata CLI transactions.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Aliases:                    []string{"m"},
		Short:                      "Transaction commands for the metadata module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		WriteMetadataScopeCmd(),
		RemoveMetadataScopeCmd(),

		BindOsLocatorCmd(),
		RemoveOsLocatorCmd(),
		ModifyOsLocatorCmd(),

		WriteScopeSpecificationCmd(),
		RemoveScopeSpecificationCmd(),

		WriteContractSpecificationCmd(),
		RemoveContractSpecificationCmd(),

		WriteRecordSpecificationCmd(),
		RemoveRecordSpecificationCmd(),
	)

	return txCmd
}

// WriteMetadataScopeCmd creates a command for adding or updating a metadata scope.
func WriteMetadataScopeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "write-scope [scope-id] [spec-id] [owner-addresses] [data-access] [value-owner-address]",
		Short: "Add/Update a metadata scope to the provenance blockchain",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var scopeID types.MetadataAddress
			scopeID, err = types.MetadataAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			var specID types.MetadataAddress
			specID, err = types.MetadataAddressFromBech32(args[1])
			if err != nil {
				return err
			}

			ownerAddresses := strings.Split(args[2], ",")
			owners := make([]types.Party, len(ownerAddresses))
			for i, ownerAddr := range ownerAddresses {
				owners[i] = types.Party{Address: ownerAddr, Role: types.PartyType_PARTY_TYPE_OWNER}
			}
			dataAccess := strings.Split(args[3], ",")
			valueOwnerAddress := args[4]

			signers, err := parseSigners(cmd, &clientCtx)
			if err != nil {
				return err
			}

			scope := *types.NewScope(
				scopeID,
				specID,
				owners,
				dataAccess,
				valueOwnerAddress)

			msg := types.NewMsgWriteScopeRequest(scope, signers)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	addSignerFlagCmd(cmd)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// RemoveMetadataScopeCmd creates a command for removing a scope.
func RemoveMetadataScopeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-scope [scope-id]",
		Short: "Remove a metadata scope to the provenance blockchain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var scopeID types.MetadataAddress
			scopeID, err = types.MetadataAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			signers, err := parseSigners(cmd, &clientCtx)
			if err != nil {
				return err
			}

			msg := *types.NewMsgDeleteScopeRequest(scopeID, signers)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	addSignerFlagCmd(cmd)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// BindOsLocatorCmd creates a command for binding an owner to uri in the object store.
func BindOsLocatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bind-locator [owner] [uri]",
		Short: "Bind a uri to an owner address on the provenance blockchain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			if _, errAddr := sdk.AccAddressFromBech32(args[0]); errAddr != nil {
				fmt.Printf("failed to bind locator for a given owner address, invalid address: %s\n", args[0])
				return fmt.Errorf("invalid address: %w", errAddr)
			}

			objectStoreLocator := types.ObjectStoreLocator{
				LocatorUri: args[1], Owner: args[0],
			}

			addOSLocator := *types.NewMsgBindOSLocatorRequest(objectStoreLocator)
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &addOSLocator)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// RemoveOsLocatorCmd creates a command for removing an object store locator entry.
func RemoveOsLocatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-locator [owner] [uri]",
		Short: "Remove an os locator already associated owner address on the provenance blockchain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			if _, errAddr := sdk.AccAddressFromBech32(args[0]); errAddr != nil {
				fmt.Printf("failed to remove locator for a given owner address, invalid address: %s\n", args[0])
				return fmt.Errorf("invalid address: %w", errAddr)
			}

			objectStoreLocator := types.ObjectStoreLocator{
				LocatorUri: args[1], Owner: args[0],
			}

			deleteOSLocator := *types.NewMsgDeleteOSLocatorRequest(objectStoreLocator)
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &deleteOSLocator)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// ModifyOsLocatorCmd creates a command to modify the object store locator uri for an owner.
func ModifyOsLocatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modify-locator [owner] [uri]",
		Short: "Modify a uri already associated owner address on the provenance blockchain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			if _, errAddr := sdk.AccAddressFromBech32(args[0]); errAddr != nil {
				fmt.Printf("failed to add locator for a given owner address, invalid address: %s\n", args[0])
				return fmt.Errorf("invalid address: %w", errAddr)
			}
			if err != nil {
				fmt.Printf("Invalid uuid for scope id: %s", args[0])
				return err
			}

			objectStoreLocator := types.ObjectStoreLocator{
				LocatorUri: args[1], Owner: args[0],
			}

			modifyOSLocator := *types.NewMsgModifyOSLocatorRequest(objectStoreLocator)
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &modifyOSLocator)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// WriteScopeSpecificationCmd creates a command for adding scope specificiation
func WriteScopeSpecificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "write-scope-specification [specification-id] [owner-addresses] [responsible-parties] [contract-specification-ids] [description-name] [description] [website-url] [icon-url]",
		Short: "Add/Update metadata scope specification to the provenance blockchain",
		Args:  cobra.RangeArgs(4, 8),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			specificationID, err := types.MetadataAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			cSpecIds := strings.Split(args[3], ",")
			contractSpecIds := make([]types.MetadataAddress, len(cSpecIds))
			for i, cid := range cSpecIds {
				contractSpecIds[i], err = types.MetadataAddressFromBech32(cid)
				if err != nil {
					return err
				}
			}

			signers, err := parseSigners(cmd, &clientCtx)
			if err != nil {
				return err
			}

			scopeSpec := types.ScopeSpecification{
				SpecificationId: specificationID,
				OwnerAddresses:  strings.Split(args[1], ","),
				Description:     parseDescription(args[4:]),
				PartiesInvolved: parsePartyTypes(args[2]),
				ContractSpecIds: contractSpecIds,
			}

			msg := types.NewMsgWriteScopeSpecificationRequest(scopeSpec, signers)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	addSignerFlagCmd(cmd)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// WriteContractSpecificationCmd creates a command to add/update contract specifications
func WriteContractSpecificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "write-contract-specification [contractspec-id] [owners] [parties-involved] [source-value] [classname] [description-name] [description] [website-url] [icon-url]",
		Short: "Add/Update metadata contract specification on the provenance blockchain",
		Long: `Add/Update metadata contract specification on the provenance blockchain
[contractspec-id] - contract specification metaaddress
[owners] - comma delimited list of bech32 owner addresses
[parties-involved] - comma delimited list of party types.  Accepted values: originator,servicer,investor,custodian,owner,affiliate,omnibus,provenance
[source-value] - source identifier of type hash or resourceid
[classname] - Name of contract specification
[description-name] - optional- description name identifier
[description] - optional - description text
[website-url] - optional - address of website
[icon-url] - optional - address to a image to be used as an icon
		`,
		Args: cobra.RangeArgs(5, 9),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			var specificationID types.MetadataAddress
			specificationID, err = types.MetadataAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			partiesInvolved := parsePartyTypes(args[2])
			description := parseDescription(args[5:])
			contractSpecification := types.ContractSpecification{SpecificationId: specificationID,
				Description:     description,
				OwnerAddresses:  strings.Split(args[1], ","),
				PartiesInvolved: partiesInvolved,
				ClassName:       args[4],
			}
			sourceValue := args[3]
			var recordID types.MetadataAddress
			recordID, err = types.MetadataAddressFromBech32(sourceValue)
			if err != nil {
				contractSpecification.Source = &types.ContractSpecification_Hash{
					Hash: sourceValue,
				}
			} else {
				contractSpecification.Source = &types.ContractSpecification_ResourceId{
					ResourceId: recordID,
				}
			}

			signers, err := parseSigners(cmd, &clientCtx)
			if err != nil {
				return err
			}

			msg := types.NewMsgWriteContractSpecificationRequest(contractSpecification, signers)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	addSignerFlagCmd(cmd)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func WriteRecordSpecificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "write-record-specification [specification-id] [name] [input-specifications] [type-name] [result-types] [responsible-parties]",
		Short: "Add/Update metadata record specification to the provenance blockchain",
		Long: fmt.Sprintf(`Add/Update metadata record specification to the provenance blockchain.
[specification-id] - record specification metaaddress
[name] - record name
[input-specifications] - semi-colon delimited list of input specifications <name>,<type-name>,<source-value>
[type-name] - contract specification type name
[result-types] - result definition type.  Accepted values: proposed,record,record_list
[responsible-parties] - comma delimited list of party types.  Accepted values: originator,servicer,investor,custodian,owner,affiliate,omnibus,provenance

Example: 
$ %s tx metadata recspec1qh... recordname inputname1,typename1,hashvalue;inputename2,typename2,<recordmetaaddress> record_list owner,originator

`, version.AppName),
		Args: cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			specificationID, err := types.MetadataAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			recordName := args[1]

			inputs, err := parseInputSpecification(args[2])
			if err != nil {
				return err
			}

			resultType := definitionType(args[4])
			partyTypes := parsePartyTypes(args[5])
			signers, err := parseSigners(cmd, &clientCtx)
			if err != nil {
				return err
			}

			recordSpecification := types.RecordSpecification{
				SpecificationId:    specificationID,
				Name:               recordName,
				Inputs:             inputs,
				TypeName:           args[3],
				ResultType:         resultType,
				ResponsibleParties: partyTypes,
			}

			msg := *types.NewMsgWriteRecordSpecificationRequest(recordSpecification, signers)

			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	addSignerFlagCmd(cmd)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// parseInputSpecification converts cli delimited argument and converts it to InputSpecifications
func parseInputSpecification(cliDelimitedValue string) ([]*types.InputSpecification, error) {
	delimitedInputs := strings.Split(cliDelimitedValue, ";")
	inputs := make([]*types.InputSpecification, len(delimitedInputs))
	for i, delimitedInput := range delimitedInputs {
		values := strings.Split(delimitedInput, ",")
		if len(values) != 3 {
			return nil, fmt.Errorf("invalid number of values for input specification: %v", len(values))
		}
		inputs[i] = &types.InputSpecification{
			Name:     values[0],
			TypeName: values[1],
		}
		sourceValue := values[2]
		recordID, err := types.MetadataAddressFromBech32(sourceValue)
		if err != nil {
			inputs[i].Source = &types.InputSpecification_Hash{Hash: sourceValue}
		} else {
			inputs[i].Source = &types.InputSpecification_RecordId{RecordId: recordID}
		}
	}
	return inputs, nil
}

func addSignerFlagCmd(cmd *cobra.Command) {
	cmd.Flags().String(FlagSigners, "", "comma delimited list of bech32 addresses")
}

func parseSigners(cmd *cobra.Command, client *client.Context) ([]string, error) {
	flagSet := cmd.Flags()
	if flagSet.Changed(FlagSigners) {
		signerList, _ := flagSet.GetString(FlagSigners)
		signers := strings.Split(signerList, ",")
		for _, signer := range signers {
			_, err := sdk.AccAddressFromBech32(signer)
			if err != nil {
				fmt.Printf("signer address must be a Bech32 string: %v", err)
				return nil, err
			}
		}
		return signers, nil
	}
	return []string{client.GetFromAddress().String()}, nil
}

func parsePartyTypes(delimitedPartyTypes string) []types.PartyType {
	parties := strings.Split(delimitedPartyTypes, ",")
	partyTypes := make([]types.PartyType, len(parties))
	for i, party := range parties {
		partyValue := types.PartyType_value[fmt.Sprintf("PARTY_TYPE_%s", strings.ToUpper(party))]
		partyTypes[i] = types.PartyType(partyValue)
	}
	return partyTypes
}

func definitionType(cliValue string) types.DefinitionType {
	typeValue := types.DefinitionType_value[fmt.Sprintf("DEFINITION_TYPE_%s", strings.ToUpper(cliValue))]
	return types.DefinitionType(typeValue)
}

func parseDescription(cliArgs []string) *types.Description {
	if len(cliArgs) == 0 {
		return nil
	}

	description := types.Description{}
	if len(cliArgs) >= 1 {
		description.Name = cliArgs[0]
	}
	if len(cliArgs) >= 2 {
		description.Description = cliArgs[1]
	}
	if len(cliArgs) >= 3 {
		description.WebsiteUrl = cliArgs[2]
	}
	if len(cliArgs) >= 4 {
		description.IconUrl = cliArgs[3]
	}
	return &description
}

// RemoveScopeSpecificationCmd creates a command to remove scope specification
func RemoveScopeSpecificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-scope-specification [specification-id]",
		Short: "Remove scope specification from the provenance blockchain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var specificationID types.MetadataAddress
			specificationID, err = types.MetadataAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			signers, err := parseSigners(cmd, &clientCtx)
			if err != nil {
				return err
			}

			msg := *types.NewMsgDeleteScopeSpecificationRequest(specificationID, signers)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	addSignerFlagCmd(cmd)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// RemoveContractSpecificationCmd creates a command to remove a contract specification
func RemoveContractSpecificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-contract-specification [specification-id]",
		Short: "Removes a contract specification on the provenance blockchain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var specificationID types.MetadataAddress
			specificationID, err = types.MetadataAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			signers, err := parseSigners(cmd, &clientCtx)
			if err != nil {
				return err
			}

			msg := types.NewMsgDeleteContractSpecificationRequest(specificationID, signers)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	addSignerFlagCmd(cmd)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// RemoveRecordSpecificationCmd creates a command to remove record specification
func RemoveRecordSpecificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-record-specification [specification-id]",
		Short: "Remove record specification from the provenance blockchain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var specificationID types.MetadataAddress
			specificationID, err = types.MetadataAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			if !specificationID.IsRecordSpecificationAddress() {
				return fmt.Errorf("invalid contract specification id: %s", args[0])
			}
			signers, err := parseSigners(cmd, &clientCtx)
			if err != nil {
				return err
			}
			msg := *types.NewMsgDeleteRecordSpecificationRequest(specificationID, signers)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	addSignerFlagCmd(cmd)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
