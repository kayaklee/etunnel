// func goID() int64
TEXT ·goID(SB),$0-8
        MOVQ TLS, CX
        MOVQ 0(CX)(TLS*1), AX
        MOVQ AX, ret+0(FP)
        RET
