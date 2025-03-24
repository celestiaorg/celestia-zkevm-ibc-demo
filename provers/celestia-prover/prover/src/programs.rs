//! Programs for `sp1-ics07-tendermint`.

pub use sp1_ics07_tendermint_prover::programs::SP1Program;

/// SP1 ICS07 Tendermint update client program.
pub struct UpdateClientProgram;

/// SP1 ICS07 Tendermint verify (non)membership program.
pub struct MembershipProgram;

impl SP1Program for UpdateClientProgram {
    const ELF: &'static [u8] =
        include_bytes!("../../../../solidity-ibc-eureka/target/elf-compilation/riscv32im-succinct-zkvm-elf/release/sp1-ics07-tendermint-update-client");
}

impl SP1Program for MembershipProgram {
    const ELF: &'static [u8] =
        include_bytes!("../../../../solidity-ibc-eureka/target/elf-compilation/riscv32im-succinct-zkvm-elf/release/sp1-ics07-tendermint-membership");
}
