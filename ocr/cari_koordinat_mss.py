# cari_koordinat_mss.py
import mss
import mss.tools
import time

print("Buka Roblox, pastikan kotak kata terlihat...")
print("Screenshot seluruh layar dalam 5 detik...")
time.sleep(5)

with mss.mss() as sct:
    print("Info semua monitor:")
    for i, m in enumerate(sct.monitors):
        print(f"  Monitor {i}: {m}")
    
    # Screenshot monitor utama
    shot = sct.grab(sct.monitors[1])
    mss.tools.to_png(shot.rgb, shot.size, output="fullscreen_mss.png")
    print("✅ Disimpan ke fullscreen_mss.png")