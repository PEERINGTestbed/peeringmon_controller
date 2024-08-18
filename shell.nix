{ pkgs ? import <nixpkgs> {}}:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go
  ];

  RUST_BACKTRACE = 1;
}
