"""Dump entries of the +336 lookup table found in
../README.md (Session 2). Not general-purpose: PID and BASE are hardcoded
from one specific live run against the pre-storm baseline (seed 2,
DeepDesert_1 pid 390735) and will need updating against any other process.
"""
import struct

PID = 390735
BASE = 0x75f8b6ee0000
STRIDE = 0x70

def readmem(f, addr, n):
    f.seek(addr)
    return f.read(n)

def try_string(f, addr, maxlen=64):
    if addr == 0 or addr > 0x7fffffffffff:
        return None
    try:
        raw = readmem(f, addr, maxlen)
    except Exception:
        return None
    out = []
    for b in raw:
        if b == 0:
            break
        if 32 <= b < 127:
            out.append(chr(b))
        else:
            return None
    if len(out) < 2:
        return None
    return ''.join(out)

with open(f'/proc/{PID}/mem', 'rb') as f:
    for i in range(40):
        entry_addr = BASE + i*STRIDE
        try:
            raw = readmem(f, entry_addr, STRIDE)
        except Exception as e:
            print(i, 'read failed', e)
            continue
        byte10 = raw[0x10]
        ptr50 = struct.unpack('<Q', raw[0x50:0x58])[0]
        s = try_string(f, ptr50)
        print(f'[{i:3d}] addr={hex(entry_addr)} byte+10={byte10:#04x} ptr+50={hex(ptr50)} str={s!r}')
