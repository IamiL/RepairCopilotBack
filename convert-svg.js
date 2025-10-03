const fs = require('fs');
const { createCanvas, loadImage } = require('canvas');

async function convertSvgToPng() {
  const canvas = createCanvas(30, 30);
  const ctx = canvas.getContext('2d');

  // Читаем SVG
  const svg = fs.readFileSync('files/logo.svg', 'utf8');

  // Создаем data URL из SVG
  const svgBuffer = Buffer.from(svg);
  const svgDataUrl = `data:image/svg+xml;base64,${svgBuffer.toString('base64')}`;

  // Загружаем и рисуем
  const img = await loadImage(svgDataUrl);
  ctx.drawImage(img, 0, 0);

  // Сохраняем PNG
  const buffer = canvas.toBuffer('image/png');
  fs.writeFileSync('files/logo.png', buffer);
  console.log('Конвертация завершена: files/logo.png');
}

convertSvgToPng().catch(console.error);