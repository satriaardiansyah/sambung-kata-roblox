import pyautogui
import time

print("Arahkan mouse ke pojok KIRI ATAS area teks Roblox, tunggu 5 detik...")
time.sleep(5)
x1, y1 = pyautogui.position()
print(f"Kiri atas: ({x1}, {y1})")

print("Arahkan ke pojok KANAN BAWAH, tunggu 5 detik...")
time.sleep(5)
x2, y2 = pyautogui.position()
print(f"Kanan bawah: ({x2}, {y2})")

print(f"\nMasukkan ini ke roblox_reader.py:")
print(f"REGION = ({x1}, {y1}, {x2-x1}, {y2-y1})")