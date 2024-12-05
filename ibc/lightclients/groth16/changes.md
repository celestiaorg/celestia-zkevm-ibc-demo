# Changes

Some of the work I did in order to reflect recent changes in the IBC:

- Removed genesis.go because - The [`ExportMetadata` interface function](https://github.com/cosmos/ibc-go/blob/v8.0.0/modules/core/exported/client.go#L59) has been removed from the `ClientState` interface. Core IBC will export all key/value's within the 02-client store.
- Keeper function `CheckMisbehaviourAndUpdateState` has been removed since function `UpdateClient` can now handle updating `ClientState` on `ClientMessage` type which can be any `Misbehaviour` implementations.
- VerifyClientState and VerifyClientConsensusState were removed.
- Ref: <https://github.com/cosmos/ibc/issues/1121>
- Have to add module.go in order to initialize the client in the cosmos-sdk app.
