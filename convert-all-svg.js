const fs = require('fs');
const path = require('path');
const { createCanvas, loadImage } = require('canvas');

const svgFiles = [
  'files/logo.svg',
  'files/profile-icon.svg',
  'files/directoryIcon.svg',
  'files/dotIcon.svg',
  'files/likeIcon.svg',
  'files/dislikeIcon.svg',
  'files/closeIcon.svg',
  'files/allElements.svg'
];

async function convertSvgToPng(svgPath) {
  const svg = fs.readFileSync(svgPath, 'utf8');

  // Извлекаем размеры из SVG
  const widthMatch = svg.match(/width="(\d+)"/);
  const heightMatch = svg.match(/height="(\d+)"/);
  const width = widthMatch ? parseInt(widthMatch[1]) : 24;
  const height = heightMatch ? parseInt(heightMatch[1]) : 24;

  const canvas = createCanvas(width, height);
  const ctx = canvas.getContext('2d');

  const svgBuffer = Buffer.from(svg);
  const svgDataUrl = `data:image/svg+xml;base64,${svgBuffer.toString('base64')}`;

  const img = await loadImage(svgDataUrl);
  ctx.drawImage(img, 0, 0, width, height);

  const pngPath = svgPath.replace('.svg', '.png');
  const buffer = canvas.toBuffer('image/png');
  fs.writeFileSync(pngPath, buffer);
  console.log(`✓ ${path.basename(svgPath)} -> ${path.basename(pngPath)}`);
}

async function convertAll() {
  console.log('Конвертация SVG в PNG...\n');
  for (const file of svgFiles) {
    await convertSvgToPng(file);
  }
  console.log('\nГотово!');
}

convertAll().catch(console.error);