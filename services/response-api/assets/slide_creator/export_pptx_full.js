const fs = require('fs');
const path = require('path');

/**
 * HTML -> PPTX exporter
 *
 * Modes:
 *   --mode dom    : Convert rendered HTML DOM into an *editable* PPTX (text boxes/shapes/images)
 *                  using a client-side converter injected into Playwright.
 *   --mode raster : Screenshot HTML slides into PNGs (not editable) and place full-bleed in PPTX.
 *                  Optional: overlay PPTX-native charts (bar/line/pie) on top of the raster slide.
 *   --mode auto   : Prefer dom if slide HTML exists, otherwise raster.
 *
 * Why this exists:
 *   The current "screenshot" approach preserves visuals but produces a non-editable PPTX.
 *   DOM conversion rebuilds the slide as PPTX objects so you can edit text/images in PowerPoint.
 */

async function main() {
  const args = process.argv.slice(2);
  const inputDir = getArg(args, '--in', 'output');
  const outFile = getArg(args, '--out', 'deck.pptx');
  const modeArg = (getArg(args, '--mode', 'auto') || 'auto').toLowerCase();

  const vw = toInt(getArg(args, '--vw', '1920'), 1920);
  const vh = toInt(getArg(args, '--vh', '1080'), 1080);
  const rootSelector = getArg(args, '--root', 'body > div');

  // dom-to-pptx bundle source (pick one)
  const domBundlePath = getArg(args, '--domjs', '');
  const domBundleUrl = getArg(
    args,
    '--domurl',
    'https://cdn.jsdelivr.net/npm/dom-to-pptx@latest/dist/dom-to-pptx.bundle.js'
  );

  // Raster quality scale (higher = sharper, larger files)
  const rasterScale = toNumber(getArg(args, '--scale', '2'), 2);
  const chartsArg = getArg(args, '--charts', '');

  if (!fs.existsSync(inputDir)) {
    throw new Error(`Input directory not found: ${inputDir}`);
  }

  const htmlFiles = listSlideHtmlFiles(inputDir);
  if (htmlFiles.length === 0) {
    throw new Error(`No slide-*.html files found in ${inputDir}`);
  }

  let mode = modeArg;
  if (mode === 'auto') {
    // If you have HTML slides, default to DOM conversion (editable) instead of raster.
    mode = htmlFiles.length > 0 ? 'dom' : 'raster';
  }

  const chartsPath = chartsArg || path.join(inputDir, 'charts.json');
  const chartsById = loadCharts(chartsPath);
  if (chartsById && mode !== 'raster') {
    console.warn(
      'Chart overlay is only supported in raster mode. Use --mode raster to apply PPTX-native charts.'
    );
  }

  ensureParentDir(outFile);

  if (mode === 'dom') {
    await exportDomEditable({
      inputDir,
      htmlFiles,
      outFile,
      vw,
      vh,
      rootSelector,
      domBundlePath,
      domBundleUrl,
    });
    return;
  }

  if (mode === 'raster') {
    await exportRaster({
      inputDir,
      htmlFiles,
      outFile,
      vw,
      vh,
      rootSelector,
      rasterScale,
      chartsById,
    });
    return;
  }

  throw new Error(`Unknown --mode "${modeArg}". Use: auto | dom | raster`);
}

function listSlideHtmlFiles(inputDir) {
  return fs
    .readdirSync(inputDir)
    .filter((f) => /^slide-\d+\.html$/i.test(f))
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true }));
}

function ensureParentDir(outFile) {
  const dir = path.dirname(path.resolve(outFile));
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
}

function getArg(args, key, fallback) {
  const idx = args.indexOf(key);
  if (idx === -1) return fallback;
  const val = args[idx + 1];
  if (!val || val.startsWith('--')) return fallback;
  return val;
}

function toNumber(val, fallback) {
  const n = Number(val);
  return Number.isFinite(n) ? n : fallback;
}

function toInt(val, fallback) {
  const n = Number.parseInt(String(val), 10);
  return Number.isFinite(n) ? n : fallback;
}


function loadCharts(chartsPath) {
  if (!chartsPath || !fs.existsSync(chartsPath)) return null;
  try {
    const raw = fs.readFileSync(chartsPath, 'utf-8');
    const parsed = JSON.parse(raw);
    const slides = Array.isArray(parsed) ? parsed : parsed.slides || [];
    const byId = {};
    for (const item of slides) {
      if (!item || typeof item !== 'object') continue;
      const id = Number(item.id);
      if (!Number.isFinite(id) || id <= 0) continue;
      byId[id] = item;
    }
    return Object.keys(byId).length ? byId : null;
  } catch (err) {
    console.warn(`Failed to parse charts JSON: ${chartsPath}`, err.message || err);
    return null;
  }
}

function slideIdFromFile(file) {
  const match = /slide-(\d+)\.html/i.exec(file);
  if (!match) return 0;
  return Number(match[1]);
}

function buildPptxChart(chartCfg, pptx, { vw, vh }) {
  if (!chartCfg || !chartCfg.type || !chartCfg.series) return null;

  const type = chartTypeFromString(chartCfg.type, pptx);
  if (!type) return null;

  const series = Array.isArray(chartCfg.series) ? chartCfg.series : [];
  if (series.length === 0) return null;

  const categories =
    chartCfg.categories ||
    chartCfg.labels ||
    (series[0] ? series[0].labels || series[0].categories : null);

  const maxLen = Math.max(
    0,
    ...series.map((s) => (Array.isArray(s.values) ? s.values.length : 0))
  );
  const fallbackLabels = Array.from({ length: maxLen }, (_, i) => `${i + 1}`);

  const data = series.map((s, idx) => ({
    name: s.name || `Series ${idx + 1}`,
    labels: categories || fallbackLabels,
    values: Array.isArray(s.values) ? s.values : [],
  }));

  const pos = normalizeChartPosition(chartCfg.position, { vw, vh });
  const options = {
    x: pos.x,
    y: pos.y,
    w: pos.w,
    h: pos.h,
    showLegend: chartCfg.showLegend !== false,
  };
  if (chartCfg.title) {
    options.title = chartCfg.title;
  }

  return { type, data, options };
}

function chartTypeFromString(type, pptx) {
  const t = String(type || '').toLowerCase();
  if (t === 'bar' || t === 'column') return pptx.ChartType.bar;
  if (t === 'line') return pptx.ChartType.line;
  if (t === 'pie') return pptx.ChartType.pie;
  return null;
}

function normalizeChartPosition(position, { vw, vh }) {
  const fallback = { x: 1.1, y: 1.3, w: 11.2, h: 5.3 };
  if (!position || typeof position !== 'object') return fallback;

  const units = String(position.units || '').toLowerCase();
  if (units === 'px') {
    const toIn = (val, totalPx, totalIn) =>
      Number.isFinite(val) ? (val / totalPx) * totalIn : null;
    const x = toIn(position.x, vw, 13.333);
    const y = toIn(position.y, vh, 7.5);
    const w = toIn(position.w, vw, 13.333);
    const h = toIn(position.h, vh, 7.5);
    return {
      x: x ?? fallback.x,
      y: y ?? fallback.y,
      w: w ?? fallback.w,
      h: h ?? fallback.h,
    };
  }

  const toNum = (v) => (Number.isFinite(v) ? v : null);
  return {
    x: toNum(position.x) ?? fallback.x,
    y: toNum(position.y) ?? fallback.y,
    w: toNum(position.w) ?? fallback.w,
    h: toNum(position.h) ?? fallback.h,
  };
}

async function exportRaster({ inputDir, htmlFiles, outFile, vw, vh, rootSelector, rasterScale, chartsById }) {
  const pptxgen = require('pptxgenjs');
  const { chromium } = require('playwright');

  const pptx = new pptxgen();
  pptx.layout = 'LAYOUT_WIDE';
  const outDir = path.dirname(path.resolve(outFile));

  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: vw, height: vh },
    deviceScaleFactor: Math.max(1, rasterScale),
  });
  const page = await context.newPage();

  for (let i = 0; i < htmlFiles.length; i += 1) {
    const file = htmlFiles[i];
    const slideId = slideIdFromFile(file);
    const fullPath = path.resolve(inputDir, file);
    await page.goto(`file://${fullPath}`, { waitUntil: 'load' });
    await waitForFontsAndImages(page);

    const container = await page.$(rootSelector);
    if (!container) throw new Error(`No slide root found with selector "${rootSelector}" in ${file}`);

    const box = await container.boundingBox();
    if (!box) throw new Error(`Failed to get bounding box for ${file}`);
    const slide = pptx.addSlide();

    const chartCfg = chartsById ? chartsById[slideId] : null;
    const overlayChart = !!(chartCfg && chartCfg.overlay !== false);
    const image = await page.screenshot({
      type: 'png',
      clip: {
        x: Math.max(0, Math.floor(box.x)),
        y: Math.max(0, Math.floor(box.y)),
        width: Math.max(1, Math.floor(box.width)),
        height: Math.max(1, Math.floor(box.height)),
      },
    });
    const imageName = `slide-${slideId || i + 1}.png`;
    const imagePath = path.join(outDir, imageName);
    fs.writeFileSync(imagePath, image);

    if (!chartCfg || overlayChart) {
      slide.addImage({
        data: `image/png;base64,${image.toString('base64')}` ,
        x: 0,
        y: 0,
        w: 13.333,
        h: 7.5,
      });
    }

    if (chartCfg) {
      const chart = buildPptxChart(chartCfg, pptx, { vw, vh });
      if (chart) {
        slide.addChart(chart.type, chart.data, chart.options);
      }
    }
  }

  await browser.close();
  await pptx.writeFile({ fileName: outFile });
  console.log(`Wrote ${outFile} (raster)`);
}

async function exportDomEditable({
  inputDir,
  htmlFiles,
  outFile,
  vw,
  vh,
  rootSelector,
  domBundlePath,
  domBundleUrl,
}) {
  const { chromium } = require('playwright');
  const outDir = path.dirname(path.resolve(outFile));

  // 1) Load each slide file to normalize resources (absolute <img src>, fixed root size, etc.)
  // 2) Assemble all slide roots into one "deck" page
  // 3) Inject dom-to-pptx and export multi-slide PPTX

  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: vw, height: vh },
    acceptDownloads: true,
  });
  const page = await context.newPage();

  const slideOuterHtml = [];

  for (let i = 0; i < htmlFiles.length; i += 1) {
    const file = htmlFiles[i];
    const slideId = slideIdFromFile(file) || i + 1;
    const fullPath = path.resolve(inputDir, file);
    await page.goto(`file://${fullPath}`, { waitUntil: 'load' });
    await waitForFontsAndImages(page);

    const root = await page.$(rootSelector);
    if (!root) throw new Error(`No slide root found with selector "${rootSelector}" in ${file}`);

    // Normalize: lock the slide root to a deterministic 1920x1080 (or --vw/--vh)
    // and convert any relative image src -> absolute URL.
    await page.evaluate(
      ({ rootSelector, vw, vh }) => {
        const root = document.querySelector(rootSelector);
        if (!root) return;

        // Force deterministic size (dom-to-pptx recommends building at 1920x1080)
        root.style.width = `${vw}px`;
        root.style.height = `${vh}px`;
        root.style.maxWidth = `${vw}px`;

        // Make image URLs absolute + enable CORS-friendly loading.
        root.querySelectorAll('img').forEach((img) => {
          try {
            img.setAttribute('src', img.src);
          } catch (_) {
            // ignore
          }
          if (!img.getAttribute('crossorigin')) {
            img.setAttribute('crossorigin', 'anonymous');
          }
        });
      },
      { rootSelector, vw, vh }
    );

    try {
      const box = await root.boundingBox();
      if (box) {
        const image = await page.screenshot({
          type: 'png',
          clip: {
            x: Math.max(0, Math.floor(box.x)),
            y: Math.max(0, Math.floor(box.y)),
            width: Math.max(1, Math.floor(box.width)),
            height: Math.max(1, Math.floor(box.height)),
          },
        });
        const imageName = `slide-${slideId}.png`;
        fs.writeFileSync(path.join(outDir, imageName), image);
      }
    } catch (err) {
      const message = err && err.message ? err.message : err;
      console.warn(`Failed to capture slide preview for ${file}:`, message);
    }

    const outer = await page.$eval(rootSelector, (el) => el.outerHTML);
    slideOuterHtml.push(outer);
  }

  // Build a single page containing all slides as separate DOM roots.
  const deckHtml = buildDeckHtml(slideOuterHtml, { vw, vh });
  await page.setContent(deckHtml, { waitUntil: 'load' });
  await waitForFontsAndImages(page);

  // Inject dom-to-pptx bundle
  if (domBundlePath && fs.existsSync(domBundlePath)) {
    await page.addScriptTag({ path: path.resolve(domBundlePath) });
  } else {
    // Falls back to CDN (requires internet)
    await page.addScriptTag({ url: domBundleUrl });
  }

  // Wait for the global to appear (script-tag usage exposes `domToPptx`).
  await page.waitForFunction(() => !!window.domToPptx && !!window.domToPptx.exportToPptx, null, {
    timeout: 60_000,
  });

  // Prefer capturing the browser download (avoids shipping a huge Blob through Playwright RPC)
  const outName = path.basename(outFile);

  const download = await Promise.race([
    // Trigger export + wait for download
    (async () => {
      const [dl] = await Promise.all([
        page.waitForEvent('download', { timeout: 180_000 }),
        page.evaluate(async (outName) => {
          const nodes = Array.from(document.querySelectorAll('.pptx-slide'));
          if (!nodes.length) throw new Error('No .pptx-slide nodes found');
          // skipDownload=false (default) should trigger a file download
          await window.domToPptx.exportToPptx(nodes, { fileName: outName });
        }, outName),
      ]);
      return dl;
    })(),
    // If download path fails for any reason, fall back to Blob-return mode.
    (async () => {
      // short delay to let download race win when supported
      await new Promise((r) => setTimeout(r, 1000));
      return null;
    })(),
  ]);

  if (download) {
    await download.saveAs(outFile);
    await browser.close();
    console.log(`Wrote ${outFile} (dom, editable)`);
    return;
  }

  // Fallback: get Blob back and write it
  const b64 = await page.evaluate(async (outName) => {
    const nodes = Array.from(document.querySelectorAll('.pptx-slide'));
    if (!nodes.length) throw new Error('No .pptx-slide nodes found');

    const blob = await window.domToPptx.exportToPptx(nodes, {
      fileName: outName,
      skipDownload: true,
    });

    const ab = await blob.arrayBuffer();
    const bytes = new Uint8Array(ab);

    // Convert to base64 in chunks to avoid call-stack limits.
    let binary = '';
    const chunk = 0x8000;
    for (let i = 0; i < bytes.length; i += chunk) {
      binary += String.fromCharCode.apply(null, bytes.subarray(i, i + chunk));
    }
    return btoa(binary);
  }, outName);

  fs.writeFileSync(outFile, Buffer.from(b64, 'base64'));
  await browser.close();
  console.log(`Wrote ${outFile} (dom, editable)`);
}

function buildDeckHtml(slideOuterHtml, { vw, vh }) {
  // Keep the DOM simple: each slide root is wrapped in a predictable container.
  // The wrapper is what we pass to dom-to-pptx.
  const slides = slideOuterHtml
    .map(
      (outer, i) =>
        `<div class="pptx-slide" data-slide="${i + 1}" style="width:${vw}px; height:${vh}px; overflow:hidden; position:relative; margin:0; padding:0;">${outer}</div>`
    )
    .join('\n');

  return `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=${vw}, height=${vh}, initial-scale=1" />
  </head>
  <body style="margin:0; padding:0; background:#000;">
    <div id="deck" style="display:flex; flex-direction:column; gap:0; margin:0; padding:0;">
      ${slides}
    </div>
  </body>
</html>`;
}

async function waitForFontsAndImages(page) {
  // Fonts
  try {
    await page.evaluate(() => (document.fonts ? document.fonts.ready : Promise.resolve()));
  } catch (_) {
    // ignore
  }

  // Images
  try {
    await page.waitForFunction(
      () => Array.from(document.images || []).every((img) => img.complete),
      null,
      { timeout: 60_000 }
    );
  } catch (_) {
    // ignore; we still export even if some images didn't load
  }
}

main().catch((err) => {
  console.error(err?.stack || err?.message || String(err));
  process.exit(1);
});
