from PIL import Image, ImageDraw
import os

path = r"D:\XJTU-Housing-Genius\frontend\windows\runner\resources\app_icon.ico"
os.makedirs(os.path.dirname(path), exist_ok=True)

img = Image.new('RGBA', (256, 256), (0, 0, 0, 0))
d = ImageDraw.Draw(img)
d.rounded_rectangle([8, 8, 247, 247], radius=48, fill=(79, 110, 247, 255))
d.polygon([(128, 40), (36, 116), (220, 116)], fill=(255, 255, 255, 255))
d.rectangle([(60, 116), (196, 220)], fill=(255, 255, 255, 255))
d.rectangle([(108, 156), (148, 220)], fill=(79, 110, 247, 255))
img.save(path, format='ICO')
print(f"Icon saved, size={os.path.getsize(path)} bytes")
