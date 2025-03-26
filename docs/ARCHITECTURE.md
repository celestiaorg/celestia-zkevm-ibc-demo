# Architecture

![mvp-zk-accounts](./images/mvp-zk-accounts.png)

## A Breakdown of an IBC Transfer

This section takes the diagram from above and breaks down each step during `make transfer` to help aid your understanding.

![mvp-zk-accounts](./images/mvp-zk-accounts-step-1.png)

1a -> The user submits a transfer message. This is a `ICS20LibFungibleTokenPacketData` wrapped in a `SendPacket` message. As well as who it's sending the tokens to and how much it also specifies where this packet is going to and lets the eventual receiver know where the packet came from.

1b -> The SimApp chain (mimicking Celestia) executes the transaction, checking the user's balance and then moving the funds to a locked acount. It stores a commitment to this execution in state. This is kind of like a verifiable receipt.

![mvp-zk-accounts](./images/mvp-zk-accounts-step-2.png)

2a -> Now the relayer kicks in. It listens to events that SimApp has emitted that there are pending packets ready to be sent to other chains. It queries the chain for the receipt based on a predetermined location.

2b -> The relayer needs to prove to the EVM rollup that SimApp has actually successfully executed the first part of the transfer: locking up the tokens. Proving this requires two steps: First the relayer queries a state transition proof from the prover process. This will prove the latest state root from the last trusted state root stored in the state of the ICS07 Tendermint smart contract on the EVM. Now the EVM has an up to date record of SimApp's current state (which includes the receipt). Second, the relayer asks the prover for a proof that the receipt is a merkle leaf of the state root i.e. it's part of state

2c -> The prover has a zk circuit for generating both proofs. One takes tendermint headers and uses the `SkippingVerification` algorithm to assert the latest header. The other takes IAVL merkle proofs and proves some leaf key as part of the root. These are both STARK proofs which can be processed by the smart contracts on the EVM.

2d -> The last step of the relayer is to combine these proofs and packets and submit a `MsgUpdateClient` and `MsgRecvPacket` to the EVM rollup.

![mvp-zk-accounts](./images/mvp-zk-accounts-step-3.png)

Step 3 mirrors step 2 in many ways but now in the opposite direction

3a -> The EVM executes both messages. It verifies the STARK proofs and updates it's local record of SimApp's state. It then uses the updated state to verify that the receipt that the packet refers is indeed present in SimApp's state. Once all the verification checks are passed. It mints the tokens and adds them to the account of the recipient as specified in the packet. The rollup then writes it's own respective receipt that it processed the corresponding message.

3b -> Similarly, the relayer listens for events emitted from the EVM rollup for any packets awaiting to be sent back. Upon receiving the packet to be returned, an acknowledgement of the transfer to be sent back to SimApp, it talks to the prover service to prepare the relevant proofs. While they are of different state machines and different state trees, the requests are universal: a proof of the state transition and a proof of membership. The EVM Prover Service here futher compresses the STARK proofs into groth16 proofs for SimApp's groth16 IBC Client.

3c -> The relayer then sends a `MsgUpdateClient` with the state transition proof to update SimApp's record of the Rollup's state after the point that it processed the transfer packet and wrote the receipt. The relayer also sends a `MsgAcknowledgement` which contains the membership proof of the commitment, a.k.a. the receipt alongside the details of the receipt i.e. for what transfer message are we acknowledging.

3d -> SimApp processes these two messages. It validates the proofs and if everything is in order, it removes the transfer receipt and keeps one final receipt of the acknowledgement (to prevent a later timeout message).

In the case that the EVM decided these messages were not valid it would not write the acknowledgement receipt. The relayer, tracking the time when the transfer message was sent would submit a `MsgTimeout` instead of the acknowledgement with an absence proof. This is a proof that no acknowledgement was written where the predermined path says it should be written. When SimApp receives this timeout and the corresponding absence proof, it reverses the transfer, releaseing the locked funds and returning them to the sender. This process is atomic - funds can not be unlocked if they are minted on the other chain.

If someone were to send tokens from the EVM rollup back to SimApp, the source chain of those tokens, the process would be very similar, however the actions wouldn't be to lock and mint but rather the EVM rollup would burn tokens and SimApp would unlock them.
