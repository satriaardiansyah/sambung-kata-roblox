# test_region.py
import mss
import mss.tools
import time

# Coba beberapa region sekaligus untuk perbandingan
regions = {
    "kotak_besar": (1413, 867, 238, 72),   # kotak OBAT di kanan atas
    "huruf_bawah":  (1150, 690, 80,  80),    # kotak O di "Hurufnya adalah"
    "huruf_bawah2": (1130, 670, 120, 120),   # sedikit lebih lebar
}

print("Screenshot dalam 3 detik...")
time.sleep(3)

with mss.MSS() as sct:
    for nama, r in regions.items():
        monitor = {"left": r[0], "top": r[1], "width": r[2], "height": r[3]}
        shot = sct.grab(monitor)
        mss.tools.to_png(shot.rgb, shot.size, output=f"test_{nama}.png")
        print(f"✅ test_{nama}.png disimpan")