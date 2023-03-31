(module
 (type $i32_i32_i32_i32_=>_i32 (func (param i32 i32 i32 i32) (result i32)))
 (type $i32_=>_none (func (param i32)))
 (type $none_=>_none (func))
 (type $i32_i32_=>_i32 (func (param i32 i32) (result i32)))
 (type $i32_i32_=>_none (func (param i32 i32)))
 (type $i32_=>_i32 (func (param i32) (result i32)))
 (type $none_=>_i32 (func (result i32)))
 (type $i32_i32_i32_i32_=>_none (func (param i32 i32 i32 i32)))
 (type $i32_i64_i32_=>_i32 (func (param i32 i64 i32) (result i32)))
 (import "ww" "__test" (func $github.com/wetware/ww/guest/tinygo.test (param i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "fd_write" (func $runtime.fd_write (param i32 i32 i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "clock_time_get" (func $runtime.clock_time_get (param i32 i64 i32) (result i32)))
 (import "wasi_snapshot_preview1" "proc_exit" (func $runtime.proc_exit (param i32)))
 (import "wasi_snapshot_preview1" "args_sizes_get" (func $runtime.args_sizes_get (param i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "args_get" (func $runtime.args_get (param i32 i32) (result i32)))
 (memory $0 2)
 (data (i32.const 65536) "free: invalid pointer\00\00\00\00\00\01\00\15\00\00\00realloc: invalid pointer \00\01\00\18\00\00\00out of memorypanic: panic: runtime error: nil pointer dereferenceindex out of rangeslice out of range")
 (data (i32.const 65704) "x\9c\19\f6\e8\00\01\00\00\00\00\00\ac\01\01\00\c1\82\01\00\00\00\00\00\04\00\00\00\0c\00\00\00\01\00\00\00\00\00\00\00\01\00\00\00\00\00\00\00\02")
 (table $0 3 3 funcref)
 (elem (i32.const 1) $runtime.memequal $runtime.hash32)
 (global $__stack_pointer (mut i32) (i32.const 65536))
 (export "memory" (memory $0))
 (export "malloc" (func $malloc))
 (export "free" (func $free))
 (export "calloc" (func $calloc))
 (export "realloc" (func $realloc))
 (export "_start" (func $_start))
 (func $__wasm_call_ctors
  (nop)
 )
 (func $tinygo_getCurrentStackPointer (result i32)
  (global.get $__stack_pointer)
 )
 (func $strlen (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local.set $1
   (local.get $0)
  )
  (block $label$1
   (block $label$2
    (br_if $label$2
     (i32.eqz
      (i32.and
       (local.get $0)
       (i32.const 3)
      )
     )
    )
    (br_if $label$1
     (i32.eqz
      (i32.load8_u
       (local.get $0)
      )
     )
    )
    (br_if $label$2
     (i32.eqz
      (i32.and
       (local.tee $1
        (i32.add
         (local.get $0)
         (i32.const 1)
        )
       )
       (i32.const 3)
      )
     )
    )
    (br_if $label$1
     (i32.eqz
      (i32.load8_u
       (local.get $1)
      )
     )
    )
    (br_if $label$2
     (i32.eqz
      (i32.and
       (local.tee $1
        (i32.add
         (local.get $0)
         (i32.const 2)
        )
       )
       (i32.const 3)
      )
     )
    )
    (br_if $label$1
     (i32.eqz
      (i32.load8_u
       (local.get $1)
      )
     )
    )
    (br_if $label$2
     (i32.eqz
      (i32.and
       (local.tee $1
        (i32.add
         (local.get $0)
         (i32.const 3)
        )
       )
       (i32.const 3)
      )
     )
    )
    (br_if $label$1
     (i32.eqz
      (i32.load8_u
       (local.get $1)
      )
     )
    )
    (local.set $1
     (i32.add
      (local.get $0)
      (i32.const 4)
     )
    )
   )
   (local.set $1
    (i32.sub
     (local.get $1)
     (i32.const 5)
    )
   )
   (loop $label$3
    (local.set $2
     (i32.add
      (local.get $1)
      (i32.const 5)
     )
    )
    (local.set $1
     (i32.add
      (local.get $1)
      (i32.const 4)
     )
    )
    (br_if $label$3
     (i32.eqz
      (i32.and
       (i32.and
        (i32.xor
         (local.tee $2
          (i32.load
           (local.get $2)
          )
         )
         (i32.const -1)
        )
        (i32.sub
         (local.get $2)
         (i32.const 16843009)
        )
       )
       (i32.const -2139062144)
      )
     )
    )
   )
   (loop $label$4
    (br_if $label$4
     (i32.load8_u
      (local.tee $1
       (i32.add
        (local.get $1)
        (i32.const 1)
       )
      )
     )
    )
   )
  )
  (i32.sub
   (local.get $1)
   (local.get $0)
  )
 )
 (func $runtime.memequal (param $0 i32) (param $1 i32) (param $2 i32) (param $3 i32) (result i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (i32.ge_u
   (local.tee $4
    (block $label$1 (result i32)
     (loop $label$2
      (drop
       (br_if $label$1
        (local.get $2)
        (i32.eq
         (local.get $2)
         (local.get $4)
        )
       )
      )
      (local.set $5
       (i32.add
        (local.get $1)
        (local.get $4)
       )
      )
      (local.set $6
       (i32.add
        (local.get $0)
        (local.get $4)
       )
      )
      (local.set $4
       (i32.add
        (local.get $4)
        (i32.const 1)
       )
      )
      (br_if $label$2
       (i32.eq
        (i32.load8_u
         (local.get $6)
        )
        (i32.load8_u
         (local.get $5)
        )
       )
      )
     )
     (i32.sub
      (local.get $4)
      (i32.const 1)
     )
    )
   )
   (local.get $2)
  )
 )
 (func $runtime.hash32 (param $0 i32) (param $1 i32) (param $2 i32) (param $3 i32) (result i32)
  (local.set $2
   (i32.xor
    (i32.xor
     (i32.mul
      (local.get $1)
      (i32.const -962287725)
     )
     (local.get $2)
    )
    (i32.const -1130422988)
   )
  )
  (loop $label$1
   (if
    (i32.eqz
     (i32.lt_s
      (local.get $1)
      (i32.const 4)
     )
    )
    (block
     (local.set $2
      (i32.xor
       (i32.shr_u
        (local.tee $2
         (i32.mul
          (i32.add
           (i32.load align=1
            (local.get $0)
           )
           (local.get $2)
          )
          (i32.const -962287725)
         )
        )
        (i32.const 16)
       )
       (local.get $2)
      )
     )
     (local.set $1
      (i32.sub
       (local.get $1)
       (i32.const 4)
      )
     )
     (local.set $0
      (i32.add
       (local.get $0)
       (i32.const 4)
      )
     )
     (br $label$1)
    )
   )
  )
  (block $label$3
   (block $label$4
    (block $label$5
     (block $label$6
      (br_table $label$4 $label$5 $label$6 $label$3
       (i32.sub
        (local.get $1)
        (i32.const 1)
       )
      )
     )
     (local.set $2
      (i32.add
       (i32.shl
        (i32.load8_u offset=2
         (local.get $0)
        )
        (i32.const 16)
       )
       (local.get $2)
      )
     )
    )
    (local.set $2
     (i32.add
      (i32.shl
       (i32.load8_u offset=1
        (local.get $0)
       )
       (i32.const 8)
      )
      (local.get $2)
     )
    )
   )
   (local.set $2
    (i32.xor
     (i32.shr_u
      (local.tee $1
       (i32.mul
        (i32.add
         (local.get $2)
         (i32.load8_u
          (local.get $0)
         )
        )
        (i32.const -962287725)
       )
      )
      (i32.const 24)
     )
     (local.get $1)
    )
   )
  )
  (local.get $2)
 )
 (func $runtime.lookupPanic
  (call $runtime.runtimePanic
   (i32.const 65665)
   (i32.const 18)
  )
  (unreachable)
 )
 (func $runtime.runtimePanic (param $0 i32) (param $1 i32)
  (call $runtime.printstring
   (i32.const 65620)
   (i32.const 22)
  )
  (call $runtime.printstring
   (local.get $0)
   (local.get $1)
  )
  (call $runtime.printnl)
  (unreachable)
 )
 (func $runtime.slicePanic
  (call $runtime.runtimePanic
   (i32.const 65683)
   (i32.const 18)
  )
  (unreachable)
 )
 (func $runtime.printstring (param $0 i32) (param $1 i32)
  (local.set $1
   (select
    (local.get $1)
    (i32.const 0)
    (i32.gt_s
     (local.get $1)
     (i32.const 0)
    )
   )
  )
  (loop $label$1
   (if
    (local.get $1)
    (block
     (call $runtime.putchar
      (i32.load8_u
       (local.get $0)
      )
     )
     (local.set $1
      (i32.sub
       (local.get $1)
       (i32.const 1)
      )
     )
     (local.set $0
      (i32.add
       (local.get $0)
       (i32.const 1)
      )
     )
     (br $label$1)
    )
   )
  )
 )
 (func $runtime.printnl
  (call $runtime.putchar
   (i32.const 10)
  )
 )
 (func $runtime.putchar (param $0 i32)
  (local $1 i32)
  (local $2 i32)
  (if
   (i32.le_u
    (local.tee $1
     (i32.load
      (i32.const 65764)
     )
    )
    (i32.const 119)
   )
   (block
    (i32.store
     (i32.const 65764)
     (local.tee $2
      (i32.add
       (local.get $1)
       (i32.const 1)
      )
     )
    )
    (i32.store8
     (i32.add
      (local.get $1)
      (i32.const 65768)
     )
     (local.get $0)
    )
    (if
     (i32.eqz
      (i32.and
       (i32.ne
        (i32.and
         (local.get $0)
         (i32.const 255)
        )
        (i32.const 10)
       )
       (i32.ne
        (local.get $1)
        (i32.const 119)
       )
      )
     )
     (block
      (i32.store
       (i32.const 65712)
       (local.get $2)
      )
      (drop
       (call $runtime.fd_write
        (i32.const 1)
        (i32.const 65708)
        (i32.const 1)
        (i32.const 65936)
       )
      )
      (i32.store
       (i32.const 65764)
       (i32.const 0)
      )
     )
    )
    (return)
   )
  )
  (call $runtime.lookupPanic)
  (unreachable)
 )
 (func $runtime.alloc (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  (if
   (i32.eqz
    (local.get $0)
   )
   (return
    (i32.const 65928)
   )
  )
  (i64.store
   (i32.const 65904)
   (i64.add
    (i64.load
     (i32.const 65904)
    )
    (i64.extend_i32_u
     (local.get $0)
    )
   )
  )
  (i64.store
   (i32.const 65912)
   (i64.add
    (i64.load
     (i32.const 65912)
    )
    (i64.const 1)
   )
  )
  (local.set $5
   (i32.shr_u
    (i32.add
     (local.get $0)
     (i32.const 15)
    )
    (i32.const 4)
   )
  )
  (local.set $3
   (local.tee $4
    (i32.load
     (i32.const 65892)
    )
   )
  )
  (loop $label$2
   (block $label$3
    (block $label$4
     (block $label$5
      (block $label$6
       (if
        (i32.ne
         (local.get $3)
         (local.get $4)
        )
        (block
         (local.set $1
          (local.get $2)
         )
         (br $label$6)
        )
       )
       (local.set $1
        (i32.const 1)
       )
       (block $label$8
        (block $label$9
         (br_table $label$6 $label$9 $label$8
          (i32.and
           (local.get $2)
           (i32.const 255)
          )
         )
        )
        (drop
         (i32.load
          (i32.const 65932)
         )
        )
        (call $runtime.markRoots
         (call $tinygo_getCurrentStackPointer)
         (i32.const 65536)
        )
        (call $runtime.markRoots
         (i32.const 65536)
         (i32.const 66336)
        )
        (loop $label$10
         (if
          (i32.eqz
           (i32.load8_u
            (i32.const 65929)
           )
          )
          (block
           (local.set $2
            (i32.const 0)
           )
           (local.set $4
            (i32.const 0)
           )
           (local.set $1
            (i32.const 0)
           )
           (loop $label$12
            (block $label$13
             (block $label$14
              (if
               (i32.gt_u
                (i32.load
                 (i32.const 65896)
                )
                (local.get $1)
               )
               (block
                (block $label$16
                 (block $label$17
                  (block $label$18
                   (block $label$19
                    (br_table $label$16 $label$19 $label$18 $label$17 $label$13
                     (i32.and
                      (call $\28runtime.gcBlock\29.state
                       (local.get $1)
                      )
                      (i32.const 255)
                     )
                    )
                   )
                   (call $\28runtime.gcBlock\29.markFree
                    (local.get $1)
                   )
                   (i64.store
                    (i32.const 65920)
                    (i64.add
                     (i64.load
                      (i32.const 65920)
                     )
                     (i64.const 1)
                    )
                   )
                   (br $label$14)
                  )
                  (local.set $7
                   (i32.and
                    (local.get $4)
                    (i32.const 1)
                   )
                  )
                  (local.set $4
                   (i32.const 0)
                  )
                  (br_if $label$13
                   (i32.eqz
                    (local.get $7)
                   )
                  )
                  (call $\28runtime.gcBlock\29.markFree
                   (local.get $1)
                  )
                  (br $label$14)
                 )
                 (local.set $4
                  (i32.const 0)
                 )
                 (i32.store8
                  (local.tee $7
                   (i32.add
                    (i32.load
                     (i32.const 65888)
                    )
                    (i32.shr_u
                     (local.get $1)
                     (i32.const 2)
                    )
                   )
                  )
                  (i32.and
                   (i32.load8_u
                    (local.get $7)
                   )
                   (i32.xor
                    (i32.shl
                     (i32.const 2)
                     (i32.and
                      (i32.shl
                       (local.get $1)
                       (i32.const 1)
                      )
                      (i32.const 6)
                     )
                    )
                    (i32.const -1)
                   )
                  )
                 )
                 (br $label$13)
                )
                (local.set $2
                 (i32.add
                  (local.get $2)
                  (i32.const 16)
                 )
                )
                (br $label$13)
               )
              )
              (local.set $1
               (i32.const 2)
              )
              (br_if $label$6
               (i32.ge_u
                (local.get $2)
                (i32.div_u
                 (i32.sub
                  (i32.load
                   (i32.const 65888)
                  )
                  (i32.const 66336)
                 )
                 (i32.const 3)
                )
               )
              )
              (drop
               (call $runtime.growHeap)
              )
              (br $label$6)
             )
             (local.set $2
              (i32.add
               (local.get $2)
               (i32.const 16)
              )
             )
             (local.set $4
              (i32.const 1)
             )
            )
            (local.set $1
             (i32.add
              (local.get $1)
              (i32.const 1)
             )
            )
            (br $label$12)
           )
          )
         )
         (local.set $1
          (i32.const 0)
         )
         (i32.store8
          (i32.const 65929)
          (i32.const 0)
         )
         (local.set $2
          (i32.load
           (i32.const 65896)
          )
         )
         (loop $label$20
          (br_if $label$10
           (i32.ge_u
            (local.get $1)
            (local.get $2)
           )
          )
          (if
           (i32.eq
            (i32.and
             (call $\28runtime.gcBlock\29.state
              (local.get $1)
             )
             (i32.const 255)
            )
            (i32.const 3)
           )
           (block
            (call $runtime.startMark
             (local.get $1)
            )
            (local.set $2
             (i32.load
              (i32.const 65896)
             )
            )
           )
          )
          (local.set $1
           (i32.add
            (local.get $1)
            (i32.const 1)
           )
          )
          (br $label$20)
         )
        )
       )
       (local.set $1
        (local.get $2)
       )
       (br_if $label$5
        (i32.eqz
         (i32.and
          (call $runtime.growHeap)
          (i32.const 1)
         )
        )
       )
      )
      (if
       (i32.eq
        (i32.load
         (i32.const 65896)
        )
        (local.get $3)
       )
       (block
        (local.set $3
         (i32.const 0)
        )
        (br $label$4)
       )
      )
      (if
       (i32.and
        (call $\28runtime.gcBlock\29.state
         (local.get $3)
        )
        (i32.const 255)
       )
       (block
        (local.set $3
         (i32.add
          (local.get $3)
          (i32.const 1)
         )
        )
        (br $label$4)
       )
      )
      (local.set $2
       (i32.add
        (local.get $3)
        (i32.const 1)
       )
      )
      (if
       (i32.ne
        (local.get $5)
        (local.tee $6
         (i32.add
          (local.get $6)
          (i32.const 1)
         )
        )
       )
       (block
        (local.set $3
         (local.get $2)
        )
        (br $label$3)
       )
      )
      (i32.store
       (i32.const 65892)
       (local.get $2)
      )
      (call $\28runtime.gcBlock\29.setState
       (local.tee $2
        (i32.sub
         (local.get $2)
         (local.get $5)
        )
       )
       (i32.const 1)
      )
      (local.set $1
       (i32.add
        (i32.sub
         (local.get $3)
         (local.get $5)
        )
        (i32.const 2)
       )
      )
      (loop $label$25
       (if
        (i32.eqz
         (i32.eq
          (local.get $1)
          (i32.load
           (i32.const 65892)
          )
         )
        )
        (block
         (call $\28runtime.gcBlock\29.setState
          (local.get $1)
          (i32.const 2)
         )
         (local.set $1
          (i32.add
           (local.get $1)
           (i32.const 1)
          )
         )
         (br $label$25)
        )
       )
      )
      (memory.fill
       (local.tee $1
        (i32.add
         (i32.shl
          (local.get $2)
          (i32.const 4)
         )
         (i32.const 66336)
        )
       )
       (i32.const 0)
       (local.get $0)
      )
      (return
       (local.get $1)
      )
     )
     (call $runtime.runtimePanic
      (i32.const 65600)
      (i32.const 13)
     )
     (unreachable)
    )
    (local.set $6
     (i32.const 0)
    )
   )
   (local.set $4
    (i32.load
     (i32.const 65892)
    )
   )
   (local.set $2
    (local.get $1)
   )
   (br $label$2)
  )
 )
 (func $runtime.markRoots (param $0 i32) (param $1 i32)
  (local $2 i32)
  (loop $label$1
   (if
    (i32.eqz
     (i32.ge_u
      (local.get $0)
      (local.get $1)
     )
    )
    (block
     (block $label$3
      (br_if $label$3
       (i32.lt_u
        (local.tee $2
         (i32.load
          (local.get $0)
         )
        )
        (i32.const 66336)
       )
      )
      (br_if $label$3
       (i32.ge_u
        (local.get $2)
        (i32.load
         (i32.const 65888)
        )
       )
      )
      (br_if $label$3
       (i32.eqz
        (i32.and
         (call $\28runtime.gcBlock\29.state
          (local.tee $2
           (i32.shr_u
            (i32.sub
             (local.get $2)
             (i32.const 66336)
            )
            (i32.const 4)
           )
          )
         )
         (i32.const 255)
        )
       )
      )
      (br_if $label$3
       (i32.eq
        (i32.and
         (call $\28runtime.gcBlock\29.state
          (local.tee $2
           (call $\28runtime.gcBlock\29.findHead
            (local.get $2)
           )
          )
         )
         (i32.const 255)
        )
        (i32.const 3)
       )
      )
      (call $runtime.startMark
       (local.get $2)
      )
     )
     (local.set $0
      (i32.add
       (local.get $0)
       (i32.const 4)
      )
     )
     (br $label$1)
    )
   )
  )
 )
 (func $\28runtime.gcBlock\29.state (param $0 i32) (result i32)
  (i32.and
   (i32.shr_u
    (i32.load8_u
     (i32.add
      (i32.load
       (i32.const 65888)
      )
      (i32.shr_u
       (local.get $0)
       (i32.const 2)
      )
     )
    )
    (i32.and
     (i32.shl
      (local.get $0)
      (i32.const 1)
     )
     (i32.const 6)
    )
   )
   (i32.const 3)
  )
 )
 (func $\28runtime.gcBlock\29.markFree (param $0 i32)
  (local $1 i32)
  (i32.store8
   (local.tee $1
    (i32.add
     (i32.load
      (i32.const 65888)
     )
     (i32.shr_u
      (local.get $0)
      (i32.const 2)
     )
    )
   )
   (i32.and
    (i32.load8_u
     (local.get $1)
    )
    (i32.xor
     (i32.shl
      (i32.const 3)
      (i32.and
       (i32.shl
        (local.get $0)
        (i32.const 1)
       )
       (i32.const 6)
      )
     )
     (i32.const -1)
    )
   )
  )
 )
 (func $runtime.growHeap (result i32)
  (local $0 i32)
  (local $1 i32)
  (local $2 i32)
  (if
   (local.tee $1
    (i32.ne
     (memory.grow
      (memory.size)
     )
     (i32.const -1)
    )
   )
   (block
    (local.set $0
     (memory.size)
    )
    (local.set $2
     (i32.load
      (i32.const 65760)
     )
    )
    (i32.store
     (i32.const 65760)
     (i32.shl
      (local.get $0)
      (i32.const 16)
     )
    )
    (local.set $0
     (i32.load
      (i32.const 65888)
     )
    )
    (call $runtime.calculateHeapAddresses)
    (memory.copy
     (i32.load
      (i32.const 65888)
     )
     (local.get $0)
     (i32.sub
      (local.get $2)
      (local.get $0)
     )
    )
   )
  )
  (local.get $1)
 )
 (func $runtime.startMark (param $0 i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  (local $8 i32)
  (global.set $__stack_pointer
   (local.tee $3
    (i32.add
     (global.get $__stack_pointer)
     (i32.const -64)
    )
   )
  )
  (memory.fill
   (i32.add
    (local.get $3)
    (i32.const 4)
   )
   (i32.const 0)
   (i32.const 60)
  )
  (i32.store
   (local.get $3)
   (local.get $0)
  )
  (call $\28runtime.gcBlock\29.setState
   (local.get $0)
   (i32.const 3)
  )
  (local.set $2
   (i32.const 1)
  )
  (block $label$1
   (loop $label$2
    (if
     (i32.gt_s
      (local.get $2)
      (i32.const 0)
     )
     (block
      (br_if $label$1
       (i32.gt_u
        (local.tee $2
         (i32.sub
          (local.get $2)
          (i32.const 1)
         )
        )
        (i32.const 15)
       )
      )
      (local.set $0
       (i32.shl
        (local.tee $1
         (i32.load
          (i32.add
           (local.get $3)
           (i32.shl
            (local.get $2)
            (i32.const 2)
           )
          )
         )
        )
        (i32.const 4)
       )
      )
      (block $label$4
       (block $label$5
        (br_table $label$5 $label$4 $label$5 $label$4
         (i32.sub
          (i32.and
           (call $\28runtime.gcBlock\29.state
            (local.get $1)
           )
           (i32.const 255)
          )
          (i32.const 1)
         )
        )
       )
       (local.set $1
        (i32.add
         (local.get $1)
         (i32.const 1)
        )
       )
      )
      (local.set $5
       (i32.add
        (local.get $0)
        (i32.const 66336)
       )
      )
      (local.set $6
       (i32.sub
        (local.tee $4
         (i32.shl
          (local.get $1)
          (i32.const 4)
         )
        )
        (local.get $0)
       )
      )
      (local.set $4
       (i32.add
        (local.get $4)
        (i32.const 66336)
       )
      )
      (local.set $7
       (i32.load
        (i32.const 65888)
       )
      )
      (loop $label$6
       (block $label$7
        (local.set $0
         (local.get $6)
        )
        (br_if $label$7
         (i32.ge_u
          (local.get $4)
          (local.get $7)
         )
        )
        (local.set $6
         (i32.add
          (local.get $0)
          (i32.const 16)
         )
        )
        (local.set $4
         (i32.add
          (local.get $4)
          (i32.const 16)
         )
        )
        (local.set $8
         (call $\28runtime.gcBlock\29.state
          (local.get $1)
         )
        )
        (local.set $1
         (i32.add
          (local.get $1)
          (i32.const 1)
         )
        )
        (br_if $label$6
         (i32.eq
          (i32.and
           (local.get $8)
           (i32.const 255)
          )
          (i32.const 2)
         )
        )
       )
      )
      (loop $label$8
       (br_if $label$2
        (i32.eqz
         (local.get $0)
        )
       )
       (block $label$9
        (br_if $label$9
         (i32.lt_u
          (local.tee $1
           (i32.load
            (local.get $5)
           )
          )
          (i32.const 66336)
         )
        )
        (br_if $label$9
         (i32.ge_u
          (local.get $1)
          (i32.load
           (i32.const 65888)
          )
         )
        )
        (br_if $label$9
         (i32.eqz
          (i32.and
           (call $\28runtime.gcBlock\29.state
            (local.tee $1
             (i32.shr_u
              (i32.sub
               (local.get $1)
               (i32.const 66336)
              )
              (i32.const 4)
             )
            )
           )
           (i32.const 255)
          )
         )
        )
        (br_if $label$9
         (i32.eq
          (i32.and
           (call $\28runtime.gcBlock\29.state
            (local.tee $1
             (call $\28runtime.gcBlock\29.findHead
              (local.get $1)
             )
            )
           )
           (i32.const 255)
          )
          (i32.const 3)
         )
        )
        (call $\28runtime.gcBlock\29.setState
         (local.get $1)
         (i32.const 3)
        )
        (if
         (i32.eq
          (local.get $2)
          (i32.const 16)
         )
         (block
          (i32.store8
           (i32.const 65929)
           (i32.const 1)
          )
          (local.set $2
           (i32.const 16)
          )
          (br $label$9)
         )
        )
        (br_if $label$1
         (i32.gt_u
          (local.get $2)
          (i32.const 15)
         )
        )
        (i32.store
         (i32.add
          (local.get $3)
          (i32.shl
           (local.get $2)
           (i32.const 2)
          )
         )
         (local.get $1)
        )
        (local.set $2
         (i32.add
          (local.get $2)
          (i32.const 1)
         )
        )
       )
       (local.set $0
        (i32.sub
         (local.get $0)
         (i32.const 4)
        )
       )
       (local.set $5
        (i32.add
         (local.get $5)
         (i32.const 4)
        )
       )
       (br $label$8)
      )
     )
    )
   )
   (global.set $__stack_pointer
    (i32.sub
     (local.get $3)
     (i32.const -64)
    )
   )
   (return)
  )
  (call $runtime.lookupPanic)
  (unreachable)
 )
 (func $\28runtime.gcBlock\29.setState (param $0 i32) (param $1 i32)
  (local $2 i32)
  (i32.store8
   (local.tee $2
    (i32.add
     (i32.load
      (i32.const 65888)
     )
     (i32.shr_u
      (local.get $0)
      (i32.const 2)
     )
    )
   )
   (i32.or
    (i32.load8_u
     (local.get $2)
    )
    (i32.shl
     (local.get $1)
     (i32.and
      (i32.shl
       (local.get $0)
       (i32.const 1)
      )
      (i32.const 6)
     )
    )
   )
  )
 )
 (func $runtime.nilPanic
  (call $runtime.runtimePanic
   (i32.const 65642)
   (i32.const 23)
  )
  (unreachable)
 )
 (func $runtime.calculateHeapAddresses
  (local $0 i32)
  (i32.store
   (i32.const 65888)
   (local.tee $0
    (i32.sub
     (local.tee $0
      (i32.load
       (i32.const 65760)
      )
     )
     (i32.div_u
      (i32.sub
       (local.get $0)
       (i32.const 66272)
      )
      (i32.const 65)
     )
    )
   )
  )
  (i32.store
   (i32.const 65896)
   (i32.shr_u
    (i32.sub
     (local.get $0)
     (i32.const 66336)
    )
    (i32.const 4)
   )
  )
 )
 (func $\28runtime.gcBlock\29.findHead (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (loop $label$1
   (local.set $1
    (call $\28runtime.gcBlock\29.state
     (local.get $0)
    )
   )
   (local.set $0
    (local.tee $2
     (i32.sub
      (local.get $0)
      (i32.const 1)
     )
    )
   )
   (br_if $label$1
    (i32.eq
     (i32.and
      (local.get $1)
      (i32.const 255)
     )
     (i32.const 2)
    )
   )
  )
  (i32.add
   (local.get $2)
   (i32.const 1)
  )
 )
 (func $malloc (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  (global.set $__stack_pointer
   (local.tee $1
    (i32.sub
     (global.get $__stack_pointer)
     (i32.const 32)
    )
   )
  )
  (i32.store offset=20
   (local.get $1)
   (i32.const 2)
  )
  (local.set $3
   (i32.load
    (i32.const 65932)
   )
  )
  (i32.store
   (i32.const 65932)
   (i32.add
    (local.get $1)
    (i32.const 16)
   )
  )
  (i32.store offset=16
   (local.get $1)
   (local.get $3)
  )
  (block $label$1
   (if
    (local.get $0)
    (block
     (br_if $label$1
      (i32.lt_s
       (local.get $0)
       (i32.const 0)
      )
     )
     (i32.store offset=24
      (local.get $1)
      (local.tee $2
       (call $runtime.alloc
        (local.get $0)
       )
      )
     )
     (i32.store offset=28
      (local.get $1)
      (local.get $2)
     )
     (i32.store offset=8
      (local.get $1)
      (local.get $0)
     )
     (i32.store offset=4
      (local.get $1)
      (local.get $0)
     )
     (i32.store
      (local.get $1)
      (local.get $2)
     )
     (i32.store offset=12
      (local.get $1)
      (local.get $2)
     )
     (call $runtime.hashmapBinarySet
      (i32.add
       (local.get $1)
       (i32.const 12)
      )
      (local.get $1)
     )
    )
   )
   (i32.store
    (i32.const 65932)
    (local.get $3)
   )
   (global.set $__stack_pointer
    (i32.add
     (local.get $1)
     (i32.const 32)
    )
   )
   (return
    (local.get $2)
   )
  )
  (call $runtime.slicePanic)
  (unreachable)
 )
 (func $runtime.hashmapBinarySet (param $0 i32) (param $1 i32)
  (call $runtime.hashmapSet
   (i32.const 65716)
   (local.get $0)
   (local.get $1)
   (call $runtime.hash32
    (local.get $0)
    (i32.load
     (i32.const 65728)
    )
    (i32.load
     (i32.const 65720)
    )
    (local.get $0)
   )
  )
 )
 (func $free (param $0 i32)
  (local $1 i32)
  (global.set $__stack_pointer
   (local.tee $1
    (i32.sub
     (global.get $__stack_pointer)
     (i32.const 16)
    )
   )
  )
  (block $label$1
   (if
    (local.get $0)
    (block
     (i32.store offset=12
      (local.get $1)
      (local.get $0)
     )
     (br_if $label$1
      (i32.eqz
       (i32.and
        (call $runtime.hashmapBinaryGet
         (i32.add
          (local.get $1)
          (i32.const 12)
         )
         (local.get $1)
        )
        (i32.const 1)
       )
      )
     )
     (i32.store
      (local.get $1)
      (local.get $0)
     )
     (call $runtime.hashmapBinaryDelete
      (local.get $1)
     )
    )
   )
   (global.set $__stack_pointer
    (i32.add
     (local.get $1)
     (i32.const 16)
    )
   )
   (return)
  )
  (call $runtime._panic
   (i32.const 65560)
  )
  (unreachable)
 )
 (func $runtime.hashmapBinaryGet (param $0 i32) (param $1 i32) (result i32)
  (call $runtime.hashmapGet
   (i32.const 65716)
   (local.get $0)
   (local.get $1)
   (call $runtime.hash32
    (local.get $0)
    (i32.load
     (i32.const 65728)
    )
    (i32.load
     (i32.const 65720)
    )
    (local.get $0)
   )
  )
 )
 (func $runtime.hashmapBinaryDelete (param $0 i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  (local $8 i32)
  (local $9 i32)
  (global.set $__stack_pointer
   (local.tee $1
    (i32.sub
     (global.get $__stack_pointer)
     (i32.const 32)
    )
   )
  )
  (i64.store
   (i32.add
    (local.get $1)
    (i32.const 24)
   )
   (i64.const 0)
  )
  (i64.store offset=16
   (local.get $1)
   (i64.const 0)
  )
  (i32.store offset=4
   (local.get $1)
   (i32.const 6)
  )
  (local.set $6
   (i32.load
    (i32.const 65932)
   )
  )
  (i32.store
   (i32.const 65932)
   (local.get $1)
  )
  (i32.store
   (local.get $1)
   (local.get $6)
  )
  (local.set $3
   (call $runtime.hash32
    (local.get $0)
    (local.tee $2
     (i32.load
      (i32.const 65728)
     )
    )
    (i32.load
     (i32.const 65720)
    )
    (i32.const 0)
   )
  )
  (i32.store offset=8
   (local.get $1)
   (local.tee $4
    (i32.load
     (i32.const 65716)
    )
   )
  )
  (local.set $7
   (select
    (local.tee $5
     (i32.shr_u
      (local.get $3)
      (i32.const 24)
     )
    )
    (i32.const 1)
    (local.get $5)
   )
  )
  (local.set $2
   (i32.add
    (local.get $4)
    (i32.mul
     (i32.add
      (i32.shl
       (i32.add
        (local.get $2)
        (i32.load
         (i32.const 65732)
        )
       )
       (i32.const 3)
      )
      (i32.const 12)
     )
     (i32.and
      (local.get $3)
      (select
       (i32.const -1)
       (i32.xor
        (i32.shl
         (i32.const -1)
         (local.tee $2
          (i32.load8_u
           (i32.const 65736)
          )
         )
        )
        (i32.const -1)
       )
       (i32.gt_u
        (local.get $2)
        (i32.const 31)
       )
      )
     )
    )
   )
  )
  (block $label$1
   (loop $label$2
    (i32.store offset=12
     (local.get $1)
     (local.get $2)
    )
    (i32.store offset=16
     (local.get $1)
     (local.get $2)
    )
    (br_if $label$1
     (i32.eqz
      (local.get $2)
     )
    )
    (local.set $3
     (i32.const 0)
    )
    (block $label$3
     (loop $label$4
      (if
       (i32.ne
        (local.get $3)
        (i32.const 8)
       )
       (block
        (block $label$6
         (br_if $label$6
          (i32.ne
           (i32.load8_u
            (local.tee $8
             (i32.add
              (local.get $2)
              (local.get $3)
             )
            )
           )
           (local.get $7)
          )
         )
         (local.set $5
          (i32.load
           (i32.const 65728)
          )
         )
         (i32.store offset=20
          (local.get $1)
          (local.tee $9
           (i32.load
            (i32.const 65740)
           )
          )
         )
         (i32.store offset=24
          (local.get $1)
          (local.tee $4
           (i32.load
            (i32.const 65744)
           )
          )
         )
         (br_if $label$3
          (i32.eqz
           (local.get $4)
          )
         )
         (br_if $label$6
          (i32.eqz
           (i32.and
            (call_indirect (type $i32_i32_i32_i32_=>_i32)
             (local.get $0)
             (i32.add
              (i32.add
               (i32.mul
                (local.get $3)
                (local.get $5)
               )
               (local.get $2)
              )
              (i32.const 12)
             )
             (local.get $5)
             (local.get $9)
             (local.get $4)
            )
            (i32.const 1)
           )
          )
         )
         (i32.store8
          (local.get $8)
          (i32.const 0)
         )
         (i32.store
          (i32.const 65724)
          (i32.sub
           (i32.load
            (i32.const 65724)
           )
           (i32.const 1)
          )
         )
         (br $label$1)
        )
        (local.set $3
         (i32.add
          (local.get $3)
          (i32.const 1)
         )
        )
        (br $label$4)
       )
      )
     )
     (i32.store offset=28
      (local.get $1)
      (local.tee $2
       (i32.load offset=8
        (local.get $2)
       )
      )
     )
     (br $label$2)
    )
   )
   (call $runtime.nilPanic)
   (unreachable)
  )
  (i32.store
   (i32.const 65932)
   (local.get $6)
  )
  (global.set $__stack_pointer
   (i32.add
    (local.get $1)
    (i32.const 32)
   )
  )
 )
 (func $runtime._panic (param $0 i32)
  (call $runtime.printstring
   (i32.const 65613)
   (i32.const 7)
  )
  (call $runtime.printitf
   (local.get $0)
  )
  (call $runtime.printnl)
  (unreachable)
 )
 (func $calloc (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (global.set $__stack_pointer
   (local.tee $2
    (i32.sub
     (global.get $__stack_pointer)
     (i32.const 16)
    )
   )
  )
  (local.set $3
   (i32.load
    (i32.const 65932)
   )
  )
  (i32.store
   (i32.const 65932)
   (local.get $2)
  )
  (local.set $0
   (call $malloc
    (i32.mul
     (local.get $0)
     (local.get $1)
    )
   )
  )
  (i32.store
   (i32.const 65932)
   (local.get $3)
  )
  (global.set $__stack_pointer
   (i32.add
    (local.get $2)
    (i32.const 16)
   )
  )
  (local.get $0)
 )
 (func $realloc (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (global.set $__stack_pointer
   (local.tee $2
    (i32.sub
     (global.get $__stack_pointer)
     (i32.const 32)
    )
   )
  )
  (i32.store offset=20
   (local.get $2)
   (i32.const 2)
  )
  (local.set $4
   (i32.load
    (i32.const 65932)
   )
  )
  (i32.store
   (i32.const 65932)
   (i32.add
    (local.get $2)
    (i32.const 16)
   )
  )
  (i32.store offset=16
   (local.get $2)
   (local.get $4)
  )
  (block $label$1
   (block $label$2
    (block $label$3
     (if
      (i32.eqz
       (local.get $1)
      )
      (block
       (call $free
        (local.get $0)
       )
       (br $label$3)
      )
     )
     (br_if $label$2
      (i32.lt_s
       (local.get $1)
       (i32.const 0)
      )
     )
     (i32.store offset=24
      (local.get $2)
      (local.tee $3
       (call $runtime.alloc
        (local.get $1)
       )
      )
     )
     (i32.store offset=28
      (local.get $2)
      (local.get $3)
     )
     (if
      (local.get $0)
      (block
       (i32.store offset=12
        (local.get $2)
        (local.get $0)
       )
       (br_if $label$1
        (i32.eqz
         (i32.and
          (call $runtime.hashmapBinaryGet
           (i32.add
            (local.get $2)
            (i32.const 12)
           )
           (local.get $2)
          )
          (i32.const 1)
         )
        )
       )
       (memory.copy
        (local.get $3)
        (i32.load
         (local.get $2)
        )
        (select
         (local.tee $5
          (i32.load offset=4
           (local.get $2)
          )
         )
         (local.get $1)
         (i32.gt_u
          (local.get $1)
          (local.get $5)
         )
        )
       )
       (i32.store
        (local.get $2)
        (local.get $0)
       )
       (call $runtime.hashmapBinaryDelete
        (local.get $2)
       )
      )
     )
     (i32.store offset=8
      (local.get $2)
      (local.get $1)
     )
     (i32.store offset=4
      (local.get $2)
      (local.get $1)
     )
     (i32.store
      (local.get $2)
      (local.get $3)
     )
     (i32.store offset=12
      (local.get $2)
      (local.get $3)
     )
     (call $runtime.hashmapBinarySet
      (i32.add
       (local.get $2)
       (i32.const 12)
      )
      (local.get $2)
     )
    )
    (i32.store
     (i32.const 65932)
     (local.get $4)
    )
    (global.set $__stack_pointer
     (i32.add
      (local.get $2)
      (i32.const 32)
     )
    )
    (return
     (local.get $3)
    )
   )
   (call $runtime.slicePanic)
   (unreachable)
  )
  (call $runtime._panic
   (i32.const 65592)
  )
  (unreachable)
 )
 (func $_start
  (local $0 i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  (local $8 i32)
  (global.set $__stack_pointer
   (local.tee $0
    (i32.sub
     (global.get $__stack_pointer)
     (i32.const 96)
    )
   )
  )
  (i32.store offset=36
   (local.get $0)
   (i32.const 13)
  )
  (memory.fill
   (i32.add
    (local.get $0)
    (i32.const 40)
   )
   (i32.const 0)
   (i32.const 52)
  )
  (i32.store offset=32
   (local.get $0)
   (local.tee $7
    (i32.load
     (i32.const 65932)
    )
   )
  )
  (i32.store
   (i32.const 65932)
   (i32.add
    (local.get $0)
    (i32.const 32)
   )
  )
  (i32.store
   (i32.const 65760)
   (local.tee $5
    (i32.shl
     (memory.size)
     (i32.const 16)
    )
   )
  )
  (call $runtime.calculateHeapAddresses)
  (i32.store offset=44
   (local.get $0)
   (local.tee $1
    (i32.load
     (i32.const 65888)
    )
   )
  )
  (i32.store offset=40
   (local.get $0)
   (local.get $1)
  )
  (memory.fill
   (local.get $1)
   (i32.const 0)
   (i32.sub
    (local.get $5)
    (local.get $1)
   )
  )
  (i32.store
   (i32.const 65760)
   (i32.shl
    (memory.size)
    (i32.const 16)
   )
  )
  (call $__wasm_call_ctors)
  (i32.store offset=48
   (local.get $0)
   (local.tee $4
    (i32.load
     (i32.const 65944)
    )
   )
  )
  (block $label$1
   (block $label$2
    (block $label$3
     (local.set $3
      (block $label$4 (result i32)
       (if
        (local.get $4)
        (block
         (local.set $2
          (i32.load
           (i32.const 65960)
          )
         )
         (br $label$4
          (i32.load
           (i32.const 65952)
          )
         )
        )
       )
       (i32.store offset=16
        (local.get $0)
        (i32.const 0)
       )
       (i32.store offset=24
        (local.get $0)
        (i32.const 0)
       )
       (drop
        (call $runtime.args_sizes_get
         (i32.add
          (local.get $0)
          (i32.const 24)
         )
         (i32.add
          (local.get $0)
          (i32.const 16)
         )
        )
       )
       (if
        (i32.eqz
         (local.tee $2
          (i32.load offset=24
           (local.get $0)
          )
         )
        )
        (block
         (local.set $4
          (i32.const 0)
         )
         (local.set $2
          (i32.const 0)
         )
         (br $label$3)
        )
       )
       (br_if $label$1
        (i32.gt_u
         (local.get $2)
         (i32.const 1073741823)
        )
       )
       (i32.store offset=52
        (local.get $0)
        (local.tee $5
         (call $runtime.alloc
          (i32.shl
           (local.get $2)
           (i32.const 2)
          )
         )
        )
       )
       (br_if $label$1
        (i32.lt_s
         (local.tee $1
          (i32.load offset=16
           (local.get $0)
          )
         )
         (i32.const 0)
        )
       )
       (i32.store offset=56
        (local.get $0)
        (local.tee $3
         (call $runtime.alloc
          (local.get $1)
         )
        )
       )
       (i32.store offset=60
        (local.get $0)
        (local.get $3)
       )
       (br_if $label$2
        (i32.eqz
         (local.get $1)
        )
       )
       (drop
        (call $runtime.args_get
         (local.get $5)
         (local.get $3)
        )
       )
       (br_if $label$1
        (i32.gt_u
         (local.get $2)
         (i32.const 536870911)
        )
       )
       (i32.store
        (i32.const 65944)
        (local.tee $4
         (call $runtime.alloc
          (i32.shl
           (local.get $2)
           (i32.const 3)
          )
         )
        )
       )
       (i32.store
        (i32.const 65952)
        (local.get $2)
       )
       (i32.store
        (i32.const 65960)
        (local.get $2)
       )
       (i32.store offset=64
        (local.get $0)
        (local.get $4)
       )
       (local.set $3
        (local.get $4)
       )
       (local.set $6
        (local.get $2)
       )
       (loop $label$7
        (if
         (local.get $6)
         (block
          (i32.store offset=80
           (local.get $0)
           (local.tee $1
            (i32.load
             (local.get $5)
            )
           )
          )
          (i32.store offset=72
           (local.get $0)
           (local.get $1)
          )
          (i32.store offset=68
           (local.get $0)
           (local.get $1)
          )
          (i32.store
           (i32.add
            (local.get $3)
            (i32.const 4)
           )
           (local.tee $8
            (call $strlen
             (local.get $1)
            )
           )
          )
          (i32.store
           (local.get $3)
           (local.get $1)
          )
          (i32.store offset=76
           (local.get $0)
           (local.get $4)
          )
          (local.set $5
           (i32.add
            (local.get $5)
            (i32.const 4)
           )
          )
          (local.set $3
           (i32.add
            (local.get $3)
            (i32.const 8)
           )
          )
          (local.set $6
           (i32.sub
            (local.get $6)
            (i32.const 1)
           )
          )
          (br $label$7)
         )
        )
       )
       (i32.store offset=8
        (local.get $0)
        (local.get $1)
       )
       (i32.store offset=12
        (local.get $0)
        (local.get $8)
       )
       (local.get $2)
      )
     )
     (i32.store offset=84
      (local.get $0)
      (local.get $4)
     )
    )
    (i32.store
     (i32.const 66312)
     (local.get $4)
    )
    (i32.store
     (i32.const 66316)
     (local.get $3)
    )
    (i32.store
     (i32.const 66320)
     (local.get $2)
    )
    (i32.store offset=88
     (local.get $0)
     (local.get $4)
    )
    (i64.store offset=8
     (local.get $0)
     (i64.const 0)
    )
    (drop
     (call $runtime.clock_time_get
      (i32.const 0)
      (i64.const 1000)
      (i32.add
       (local.get $0)
       (i32.const 8)
      )
     )
    )
    (i32.store
     (select
      (i32.add
       (local.get $0)
       (i32.const 24)
      )
      (i32.add
       (local.get $0)
       (i32.const 16)
      )
      (i64.lt_u
       (i64.add
        (i64.div_s
         (i64.load offset=8
          (local.get $0)
         )
         (i64.const 1000000000)
        )
        (i64.const 2682288000)
       )
       (i64.const 8589934592)
      )
     )
     (i32.const 66248)
    )
    (call $runtime.proc_exit
     (call $github.com/wetware/ww/guest/tinygo.test
      (i32.const 40)
      (i32.const 2)
     )
    )
    (i32.store
     (i32.const 65932)
     (local.get $7)
    )
    (global.set $__stack_pointer
     (i32.add
      (local.get $0)
      (i32.const 96)
     )
    )
    (return)
   )
   (call $runtime.lookupPanic)
   (unreachable)
  )
  (call $runtime.slicePanic)
  (unreachable)
 )
 (func $runtime.hashmapGet (param $0 i32) (param $1 i32) (param $2 i32) (param $3 i32) (result i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  (local $8 i32)
  (local $9 i32)
  (local $10 i32)
  (local $11 i32)
  (global.set $__stack_pointer
   (local.tee $4
    (i32.sub
     (global.get $__stack_pointer)
     (i32.const 48)
    )
   )
  )
  (i32.store
   (i32.add
    (local.get $4)
    (i32.const 40)
   )
   (i32.const 0)
  )
  (i64.store offset=32
   (local.get $4)
   (i64.const 0)
  )
  (i32.store offset=12
   (local.get $4)
   (i32.const 7)
  )
  (local.set $7
   (i32.load
    (i32.const 65932)
   )
  )
  (i32.store
   (i32.const 65932)
   (i32.add
    (local.get $4)
    (i32.const 8)
   )
  )
  (i32.store offset=8
   (local.get $4)
   (local.get $7)
  )
  (i32.store offset=16
   (local.get $4)
   (local.tee $5
    (i32.load
     (local.get $0)
    )
   )
  )
  (local.set $5
   (i32.add
    (local.get $5)
    (i32.mul
     (i32.add
      (i32.shl
       (i32.add
        (i32.load offset=16
         (local.get $0)
        )
        (i32.load offset=12
         (local.get $0)
        )
       )
       (i32.const 3)
      )
      (i32.const 12)
     )
     (i32.and
      (select
       (i32.const -1)
       (i32.xor
        (i32.shl
         (i32.const -1)
         (local.tee $6
          (i32.load8_u offset=20
           (local.get $0)
          )
         )
        )
        (i32.const -1)
       )
       (i32.gt_u
        (local.get $6)
        (i32.const 31)
       )
      )
      (local.get $3)
     )
    )
   )
  )
  (local.set $9
   (select
    (local.tee $3
     (i32.shr_u
      (local.get $3)
      (i32.const 24)
     )
    )
    (i32.const 1)
    (local.get $3)
   )
  )
  (block $label$1
   (block $label$2
    (loop $label$3
     (block $label$4
      (i32.store offset=24
       (local.get $4)
       (local.get $5)
      )
      (i32.store offset=28
       (local.get $4)
       (local.get $5)
      )
      (i32.store offset=20
       (local.get $4)
       (local.get $5)
      )
      (br_if $label$4
       (i32.eqz
        (local.get $5)
       )
      )
      (local.set $3
       (i32.const 0)
      )
      (loop $label$5
       (if
        (i32.ne
         (local.get $3)
         (i32.const 8)
        )
        (block
         (block $label$7
          (br_if $label$7
           (i32.ne
            (i32.load8_u
             (i32.add
              (local.get $3)
              (local.get $5)
             )
            )
            (local.get $9)
           )
          )
          (local.set $6
           (i32.load offset=12
            (local.get $0)
           )
          )
          (local.set $10
           (i32.load offset=16
            (local.get $0)
           )
          )
          (i32.store offset=32
           (local.get $4)
           (local.tee $11
            (i32.load offset=24
             (local.get $0)
            )
           )
          )
          (i32.store offset=36
           (local.get $4)
           (local.tee $8
            (i32.load offset=28
             (local.get $0)
            )
           )
          )
          (br_if $label$2
           (i32.eqz
            (local.get $8)
           )
          )
          (br_if $label$7
           (i32.eqz
            (i32.and
             (call_indirect (type $i32_i32_i32_i32_=>_i32)
              (local.get $1)
              (i32.add
               (i32.add
                (i32.mul
                 (local.get $3)
                 (local.get $6)
                )
                (local.get $5)
               )
               (i32.const 12)
              )
              (local.get $6)
              (local.get $11)
              (local.get $8)
             )
             (i32.const 1)
            )
           )
          )
          (memory.copy
           (local.get $2)
           (i32.add
            (i32.add
             (i32.add
              (i32.mul
               (local.get $3)
               (local.get $10)
              )
              (i32.shl
               (local.get $6)
               (i32.const 3)
              )
             )
             (local.get $5)
            )
            (i32.const 12)
           )
           (i32.load offset=16
            (local.get $0)
           )
          )
          (br $label$1)
         )
         (local.set $3
          (i32.add
           (local.get $3)
           (i32.const 1)
          )
         )
         (br $label$5)
        )
       )
      )
      (i32.store offset=40
       (local.get $4)
       (local.tee $5
        (i32.load offset=8
         (local.get $5)
        )
       )
      )
      (br $label$3)
     )
    )
    (memory.fill
     (local.get $2)
     (i32.const 0)
     (i32.load offset=16
      (local.get $0)
     )
    )
    (br $label$1)
   )
   (call $runtime.nilPanic)
   (unreachable)
  )
  (i32.store
   (i32.const 65932)
   (local.get $7)
  )
  (global.set $__stack_pointer
   (i32.add
    (local.get $4)
    (i32.const 48)
   )
  )
  (i32.ne
   (local.get $5)
   (i32.const 0)
  )
 )
 (func $runtime.printitf (param $0 i32)
  (call $runtime.printstring
   (i32.load
    (local.get $0)
   )
   (i32.load offset=4
    (local.get $0)
   )
  )
 )
 (func $runtime.hashmapSet (param $0 i32) (param $1 i32) (param $2 i32) (param $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  (local $8 i32)
  (local $9 i32)
  (local $10 i32)
  (local $11 i32)
  (local $12 i32)
  (local $13 i32)
  (local $14 i32)
  (local $15 i32)
  (global.set $__stack_pointer
   (local.tee $4
    (i32.sub
     (global.get $__stack_pointer)
     (i32.const 256)
    )
   )
  )
  (i32.store offset=52
   (local.get $4)
   (i32.const 50)
  )
  (memory.fill
   (i32.add
    (local.get $4)
    (i32.const 56)
   )
   (i32.const 0)
   (i32.const 200)
  )
  (i32.store offset=48
   (local.get $4)
   (local.tee $14
    (i32.load
     (i32.const 65932)
    )
   )
  )
  (i32.store
   (i32.const 65932)
   (i32.add
    (local.get $4)
    (i32.const 48)
   )
  )
  (block $label$1
   (block $label$2
    (br_if $label$2
     (i32.eqz
      (local.get $0)
     )
    )
    (block $label$3
     (br_if $label$3
      (i32.gt_u
       (local.tee $5
        (i32.load8_u offset=20
         (local.get $0)
        )
       )
       (i32.const 29)
      )
     )
     (br_if $label$3
      (i32.le_u
       (i32.load offset=8
        (local.get $0)
       )
       (i32.shl
        (i32.const 6)
        (local.get $5)
       )
      )
     )
     (i64.store offset=24
      (local.get $4)
      (i64.const 0)
     )
     (i32.store offset=72
      (local.get $4)
      (local.tee $3
       (i32.load offset=36
        (local.get $0)
       )
      )
     )
     (i32.store offset=68
      (local.get $4)
      (local.tee $7
       (i32.load offset=32
        (local.get $0)
       )
      )
     )
     (i32.store offset=64
      (local.get $4)
      (local.tee $6
       (i32.load offset=28
        (local.get $0)
       )
      )
     )
     (i32.store offset=60
      (local.get $4)
      (local.tee $8
       (i32.load offset=24
        (local.get $0)
       )
      )
     )
     (i32.store offset=56
      (local.get $4)
      (i32.load
       (local.get $0)
      )
     )
     (i32.store offset=44
      (local.get $4)
      (local.get $3)
     )
     (i32.store offset=40
      (local.get $4)
      (local.get $7)
     )
     (i32.store offset=36
      (local.get $4)
      (local.get $6)
     )
     (i32.store offset=32
      (local.get $4)
      (local.get $8)
     )
     (i32.store offset=24
      (local.get $4)
      (i32.load offset=16
       (local.get $0)
      )
     )
     (i32.store offset=20
      (local.get $4)
      (i32.load offset=12
       (local.get $0)
      )
     )
     (i32.store
      (i32.const 65704)
      (local.tee $3
       (i32.xor
        (i32.shl
         (local.tee $3
          (i32.xor
           (i32.shr_u
            (local.tee $3
             (i32.xor
              (i32.shl
               (local.tee $3
                (i32.load
                 (i32.const 65704)
                )
               )
               (i32.const 7)
              )
              (local.get $3)
             )
            )
            (i32.const 1)
           )
           (local.get $3)
          )
         )
         (i32.const 9)
        )
        (local.get $3)
       )
      )
     )
     (i32.store offset=16
      (local.get $4)
      (i32.const 0)
     )
     (i32.store offset=12
      (local.get $4)
      (local.get $3)
     )
     (i32.store8 offset=28
      (local.get $4)
      (local.tee $3
       (i32.add
        (local.get $5)
        (i32.const 1)
       )
      )
     )
     (i32.store offset=8
      (local.get $4)
      (local.tee $3
       (call $runtime.alloc
        (i32.shl
         (i32.add
          (i32.shl
           (i32.add
            (i32.load offset=16
             (local.get $0)
            )
            (i32.load offset=12
             (local.get $0)
            )
           )
           (i32.const 3)
          )
          (i32.const 12)
         )
         (local.get $3)
        )
       )
      )
     )
     (i32.store offset=76
      (local.get $4)
      (local.get $3)
     )
     (i32.store offset=80
      (local.get $4)
      (local.tee $7
       (call $runtime.alloc
        (i32.load offset=12
         (local.get $0)
        )
       )
      )
     )
     (i32.store offset=84
      (local.get $4)
      (local.tee $13
       (call $runtime.alloc
        (i32.load offset=16
         (local.get $0)
        )
       )
      )
     )
     (local.set $3
      (i32.const 0)
     )
     (local.set $5
      (i32.const 0)
     )
     (loop $label$4
      (i32.store offset=88
       (local.get $4)
       (local.get $10)
      )
      (if
       (i32.eqz
        (local.get $10)
       )
       (block
        (i32.store offset=92
         (local.get $4)
         (local.tee $10
          (i32.load
           (local.get $0)
          )
         )
        )
        (local.set $12
         (select
          (i32.shl
           (i32.const 1)
           (local.tee $6
            (i32.load8_u offset=20
             (local.get $0)
            )
           )
          )
          (i32.const 0)
          (i32.le_u
           (local.get $6)
           (i32.const 31)
          )
         )
        )
       )
      )
      (i32.store offset=108
       (local.get $4)
       (local.get $10)
      )
      (i32.store offset=124
       (local.get $4)
       (local.get $10)
      )
      (block $label$6
       (loop $label$7
        (i32.store offset=96
         (local.get $4)
         (local.get $3)
        )
        (if
         (i32.ge_u
          (i32.and
           (local.get $5)
           (i32.const 255)
          )
          (i32.const 8)
         )
         (block
          (br_if $label$2
           (i32.eqz
            (local.get $3)
           )
          )
          (i32.store offset=100
           (local.get $4)
           (local.tee $3
            (i32.load offset=8
             (local.get $3)
            )
           )
          )
          (local.set $5
           (i32.const 0)
          )
         )
        )
        (i32.store offset=104
         (local.get $4)
         (local.get $3)
        )
        (if
         (i32.eqz
          (local.get $3)
         )
         (block
          (br_if $label$6
           (i32.ge_u
            (local.get $9)
            (local.get $12)
           )
          )
          (local.set $3
           (i32.add
            (local.get $10)
            (i32.mul
             (i32.add
              (i32.shl
               (i32.add
                (i32.load offset=16
                 (local.get $0)
                )
                (i32.load offset=12
                 (local.get $0)
                )
               )
               (i32.const 3)
              )
              (i32.const 12)
             )
             (local.get $9)
            )
           )
          )
          (local.set $9
           (i32.add
            (local.get $9)
            (i32.const 1)
           )
          )
         )
        )
        (i32.store offset=116
         (local.get $4)
         (local.get $3)
        )
        (i32.store offset=120
         (local.get $4)
         (local.get $3)
        )
        (i32.store offset=112
         (local.get $4)
         (local.get $3)
        )
        (br_if $label$2
         (i32.eqz
          (local.get $3)
         )
        )
        (if
         (i32.eqz
          (i32.load8_u
           (i32.add
            (local.get $3)
            (local.tee $8
             (i32.and
              (local.get $5)
              (i32.const 255)
             )
            )
           )
          )
         )
         (block
          (local.set $5
           (i32.add
            (local.get $5)
            (i32.const 1)
           )
          )
          (br $label$7)
         )
        )
        (memory.copy
         (local.get $7)
         (i32.add
          (i32.add
           (i32.mul
            (local.tee $6
             (i32.load offset=12
              (local.get $0)
             )
            )
            (local.get $8)
           )
           (local.get $3)
          )
          (i32.const 12)
         )
         (local.get $6)
        )
        (i32.store offset=128
         (local.get $4)
         (local.tee $11
          (i32.load
           (local.get $0)
          )
         )
        )
        (block $label$11
         (if
          (i32.eq
           (local.get $10)
           (local.get $11)
          )
          (block
           (memory.copy
            (local.get $13)
            (i32.add
             (i32.add
              (i32.add
               (i32.shl
                (local.get $6)
                (i32.const 3)
               )
               (i32.mul
                (local.tee $6
                 (i32.load offset=16
                  (local.get $0)
                 )
                )
                (local.get $8)
               )
              )
              (local.get $3)
             )
             (i32.const 12)
            )
            (local.get $6)
           )
           (local.set $5
            (i32.add
             (local.get $5)
             (i32.const 1)
            )
           )
           (br $label$11)
          )
         )
         (i32.store offset=132
          (local.get $4)
          (local.tee $11
           (i32.load offset=32
            (local.get $0)
           )
          )
         )
         (i32.store offset=136
          (local.get $4)
          (local.tee $8
           (i32.load offset=36
            (local.get $0)
           )
          )
         )
         (br_if $label$2
          (i32.eqz
           (local.get $8)
          )
         )
         (local.set $5
          (i32.add
           (local.get $5)
           (i32.const 1)
          )
         )
         (br_if $label$7
          (i32.eqz
           (i32.and
            (call $runtime.hashmapGet
             (local.get $0)
             (local.get $7)
             (local.get $13)
             (call_indirect (type $i32_i32_i32_i32_=>_i32)
              (local.get $7)
              (local.get $6)
              (i32.load offset=4
               (local.get $0)
              )
              (local.get $11)
              (local.get $8)
             )
            )
            (i32.const 1)
           )
          )
         )
        )
       )
       (i32.store offset=140
        (local.get $4)
        (local.tee $8
         (i32.load offset=40
          (local.get $4)
         )
        )
       )
       (i32.store offset=144
        (local.get $4)
        (local.tee $6
         (i32.load offset=44
          (local.get $4)
         )
        )
       )
       (br_if $label$2
        (i32.eqz
         (local.get $6)
        )
       )
       (call $runtime.hashmapSet
        (i32.add
         (local.get $4)
         (i32.const 8)
        )
        (local.get $7)
        (local.get $13)
        (call_indirect (type $i32_i32_i32_i32_=>_i32)
         (local.get $7)
         (i32.load offset=20
          (local.get $4)
         )
         (i32.load offset=12
          (local.get $4)
         )
         (local.get $8)
         (local.get $6)
        )
       )
       (br $label$4)
      )
     )
     (i32.store
      (local.get $0)
      (local.tee $3
       (i32.load offset=8
        (local.get $4)
       )
      )
     )
     (i64.store offset=4 align=4
      (local.get $0)
      (i64.load offset=12 align=4
       (local.get $4)
      )
     )
     (i64.store offset=12 align=4
      (local.get $0)
      (i64.load offset=20 align=4
       (local.get $4)
      )
     )
     (i32.store8 offset=20
      (local.get $0)
      (i32.load8_u offset=28
       (local.get $4)
      )
     )
     (i32.store offset=24
      (local.get $0)
      (local.tee $5
       (i32.load offset=32
        (local.get $4)
       )
      )
     )
     (i32.store offset=28
      (local.get $0)
      (local.tee $7
       (i32.load offset=36
        (local.get $4)
       )
      )
     )
     (i32.store offset=32
      (local.get $0)
      (local.tee $6
       (i32.load offset=40
        (local.get $4)
       )
      )
     )
     (i32.store offset=36
      (local.get $0)
      (local.tee $8
       (i32.load offset=44
        (local.get $4)
       )
      )
     )
     (i32.store offset=148
      (local.get $4)
      (local.get $3)
     )
     (i32.store offset=152
      (local.get $4)
      (local.get $5)
     )
     (i32.store offset=156
      (local.get $4)
      (local.get $7)
     )
     (i32.store offset=160
      (local.get $4)
      (local.get $6)
     )
     (i32.store offset=164
      (local.get $4)
      (local.get $8)
     )
     (i32.store offset=168
      (local.get $4)
      (local.tee $5
       (i32.load offset=32
        (local.get $0)
       )
      )
     )
     (i32.store offset=172
      (local.get $4)
      (local.tee $3
       (i32.load offset=36
        (local.get $0)
       )
      )
     )
     (br_if $label$2
      (i32.eqz
       (local.get $3)
      )
     )
     (local.set $3
      (call_indirect (type $i32_i32_i32_i32_=>_i32)
       (local.get $1)
       (i32.load offset=12
        (local.get $0)
       )
       (i32.load offset=4
        (local.get $0)
       )
       (local.get $5)
       (local.get $3)
      )
     )
     (local.set $5
      (i32.load8_u offset=20
       (local.get $0)
      )
     )
    )
    (i32.store offset=176
     (local.get $4)
     (local.tee $7
      (i32.load
       (local.get $0)
      )
     )
    )
    (local.set $9
     (i32.add
      (local.get $7)
      (i32.mul
       (i32.add
        (i32.shl
         (i32.add
          (i32.load offset=16
           (local.get $0)
          )
          (i32.load offset=12
           (local.get $0)
          )
         )
         (i32.const 3)
        )
        (i32.const 12)
       )
       (i32.and
        (select
         (i32.const -1)
         (i32.xor
          (i32.shl
           (i32.const -1)
           (local.tee $5
            (i32.and
             (local.get $5)
             (i32.const 255)
            )
           )
          )
          (i32.const -1)
         )
         (i32.gt_u
          (local.get $5)
          (i32.const 31)
         )
        )
        (local.get $3)
       )
      )
     )
    )
    (local.set $12
     (select
      (local.tee $3
       (i32.shr_u
        (local.get $3)
        (i32.const 24)
       )
      )
      (i32.const 1)
      (local.get $3)
     )
    )
    (local.set $3
     (i32.const 0)
    )
    (local.set $5
     (i32.const 0)
    )
    (local.set $8
     (i32.const 0)
    )
    (local.set $6
     (i32.const 0)
    )
    (loop $label$13
     (block $label$14
      (i32.store offset=212
       (local.get $4)
       (local.tee $7
        (local.get $9)
       )
      )
      (i32.store offset=216
       (local.get $4)
       (local.get $7)
      )
      (i32.store offset=196
       (local.get $4)
       (local.get $7)
      )
      (i32.store offset=192
       (local.get $4)
       (local.get $3)
      )
      (i32.store offset=188
       (local.get $4)
       (local.get $5)
      )
      (i32.store offset=184
       (local.get $4)
       (local.get $8)
      )
      (i32.store offset=180
       (local.get $4)
       (local.get $6)
      )
      (br_if $label$14
       (i32.eqz
        (local.get $7)
       )
      )
      (local.set $3
       (i32.const 0)
      )
      (loop $label$15
       (block $label$16
        (i32.store offset=204
         (local.get $4)
         (local.get $8)
        )
        (i32.store offset=208
         (local.get $4)
         (local.get $5)
        )
        (i32.store offset=200
         (local.get $4)
         (local.get $6)
        )
        (br_if $label$16
         (i32.eq
          (local.get $3)
          (i32.const 8)
         )
        )
        (i32.store offset=220
         (local.get $4)
         (local.tee $6
          (select
           (local.get $6)
           (local.tee $9
            (i32.add
             (local.get $3)
             (local.get $7)
            )
           )
           (local.tee $11
            (i32.or
             (i32.load8_u
              (local.get $9)
             )
             (local.get $5)
            )
           )
          )
         )
        )
        (i32.store offset=228
         (local.get $4)
         (local.tee $5
          (select
           (local.get $5)
           (local.tee $13
            (i32.add
             (i32.add
              (i32.mul
               (local.tee $10
                (i32.load offset=12
                 (local.get $0)
                )
               )
               (local.get $3)
              )
              (local.get $7)
             )
             (i32.const 12)
            )
           )
           (local.get $11)
          )
         )
        )
        (i32.store offset=224
         (local.get $4)
         (local.tee $8
          (select
           (local.get $8)
           (local.tee $15
            (i32.add
             (i32.add
              (i32.add
               (i32.mul
                (i32.load offset=16
                 (local.get $0)
                )
                (local.get $3)
               )
               (i32.shl
                (local.get $10)
                (i32.const 3)
               )
              )
              (local.get $7)
             )
             (i32.const 12)
            )
           )
           (local.get $11)
          )
         )
        )
        (block $label$17
         (br_if $label$17
          (i32.ne
           (i32.load8_u
            (local.get $9)
           )
           (local.get $12)
          )
         )
         (i32.store offset=232
          (local.get $4)
          (local.tee $11
           (i32.load offset=24
            (local.get $0)
           )
          )
         )
         (i32.store offset=236
          (local.get $4)
          (local.tee $9
           (i32.load offset=28
            (local.get $0)
           )
          )
         )
         (br_if $label$2
          (i32.eqz
           (local.get $9)
          )
         )
         (br_if $label$17
          (i32.eqz
           (i32.and
            (call_indirect (type $i32_i32_i32_i32_=>_i32)
             (local.get $1)
             (local.get $13)
             (local.get $10)
             (local.get $11)
             (local.get $9)
            )
            (i32.const 1)
           )
          )
         )
         (memory.copy
          (local.get $15)
          (local.get $2)
          (i32.load offset=16
           (local.get $0)
          )
         )
         (br $label$1)
        )
        (local.set $3
         (i32.add
          (local.get $3)
          (i32.const 1)
         )
        )
        (br $label$15)
       )
      )
      (i32.store offset=240
       (local.get $4)
       (local.tee $9
        (i32.load offset=8
         (local.get $7)
        )
       )
      )
      (local.set $3
       (local.get $7)
      )
      (br $label$13)
     )
    )
    (if
     (i32.eqz
      (local.get $5)
     )
     (block
      (local.set $5
       (call $runtime.alloc
        (i32.add
         (i32.shl
          (i32.add
           (i32.load offset=16
            (local.get $0)
           )
           (i32.load offset=12
            (local.get $0)
           )
          )
          (i32.const 3)
         )
         (i32.const 12)
        )
       )
      )
      (i32.store offset=8
       (local.get $0)
       (i32.add
        (i32.load offset=8
         (local.get $0)
        )
        (i32.const 1)
       )
      )
      (i32.store offset=248
       (local.get $4)
       (local.get $5)
      )
      (i32.store offset=252
       (local.get $4)
       (local.get $5)
      )
      (i32.store offset=244
       (local.get $4)
       (local.get $5)
      )
      (memory.copy
       (local.tee $7
        (i32.add
         (local.get $5)
         (i32.const 12)
        )
       )
       (local.get $1)
       (local.tee $6
        (i32.load offset=12
         (local.get $0)
        )
       )
      )
      (memory.copy
       (i32.add
        (local.get $7)
        (i32.shl
         (local.get $6)
         (i32.const 3)
        )
       )
       (local.get $2)
       (i32.load offset=16
        (local.get $0)
       )
      )
      (i32.store8
       (local.get $5)
       (local.get $12)
      )
      (br_if $label$2
       (i32.eqz
        (local.get $3)
       )
      )
      (i32.store offset=8
       (local.get $3)
       (local.get $5)
      )
      (br $label$1)
     )
    )
    (i32.store offset=8
     (local.get $0)
     (i32.add
      (i32.load offset=8
       (local.get $0)
      )
      (i32.const 1)
     )
    )
    (memory.copy
     (local.get $5)
     (local.get $1)
     (i32.load offset=12
      (local.get $0)
     )
    )
    (memory.copy
     (local.get $8)
     (local.get $2)
     (i32.load offset=16
      (local.get $0)
     )
    )
    (br_if $label$2
     (i32.eqz
      (local.get $6)
     )
    )
    (i32.store8
     (local.get $6)
     (local.get $12)
    )
    (br $label$1)
   )
   (call $runtime.nilPanic)
   (unreachable)
  )
  (i32.store
   (i32.const 65932)
   (local.get $14)
  )
  (global.set $__stack_pointer
   (i32.add
    (local.get $4)
    (i32.const 256)
   )
  )
 )
 ;; custom section ".debug_info", size 29604
 ;; custom section ".debug_pubtypes", size 2106
 ;; custom section ".debug_loc", size 8492
 ;; custom section ".debug_ranges", size 1440
 ;; custom section ".debug_aranges", size 80
 ;; custom section ".debug_abbrev", size 2027
 ;; custom section ".debug_line", size 11123
 ;; custom section ".debug_str", size 18173
 ;; custom section ".debug_pubnames", size 16895
 ;; custom section "producers", size 128
)
