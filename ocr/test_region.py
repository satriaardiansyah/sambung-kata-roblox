# test_region.py
import mss
import mss.tools
import time

REGION_HURUF = (1155, 700, 120, 90)

print("Screenshot dalam 3 detik, pastikan Roblox terlihat...")
time.sleep(3)

with mss.MSS() as sct:
    monitor = {
        "left":   REGION_HURUF[0],
        "top":    REGION_HURUF[1],
        "width":  REGION_HURUF[2],
        "height": REGION_HURUF[3]
    }
    shot = sct.grab(monitor)
    mss.tools.to_png(shot.rgb, shot.size, output="test_region.png")
    print("✅ Disimpan ke test_region.png")