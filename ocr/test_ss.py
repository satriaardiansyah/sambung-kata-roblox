# test_ss.py
import pyautogui
print("Screenshot...")
ss = pyautogui.screenshot()
ss.save("test.png")
print("Selesai, buka test.png")