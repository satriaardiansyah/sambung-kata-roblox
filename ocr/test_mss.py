import mss
import mss.tools

with mss.MSS() as sct:
    monitor = sct.monitors[1]
    shot = sct.grab(monitor)
    mss.tools.to_png(shot.rgb, shot.size, output="test_mss.png")
    print("✅ Berhasil!")