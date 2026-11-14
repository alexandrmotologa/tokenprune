import fs from 'fs';
import path from 'path';
import { Resvg } from '@resvg/resvg-js';

function buildLogoSvg() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="1024" height="1024">
  <defs>
    <clipPath id="squircle-clip">
      <rect x="24" y="24" width="976" height="976" rx="220" />
    </clipPath>

    <linearGradient id="emerald-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#4ade80"/>
      <stop offset="100%" stop-color="#16a34a"/>
    </linearGradient>

    <linearGradient id="emerald-optic-left" x1="100%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" stop-color="#4ade80"/>
      <stop offset="100%" stop-color="#16a34a"/>
    </linearGradient>

    <linearGradient id="beak-gold" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#fcd34d"/>
      <stop offset="100%" stop-color="#d97706"/>
    </linearGradient>

    <filter id="subtle-shadow" x="-10%" y="-10%" width="120%" height="120%">
      <feDropShadow dx="0" dy="16" stdDeviation="20" flood-color="#000000" flood-opacity="0.16" />
    </filter>
  </defs>

  <!-- Luxury White Squircle Container -->
  <rect x="24" y="24" width="976" height="976" rx="220" fill="#ffffff" stroke="#e2e8f0" stroke-width="6" />

  <g clip-path="url(#squircle-clip)">
    <g transform="translate(512, 512)" filter="url(#subtle-shadow)">

      <!-- Hexagonal Architectural Gateway Frame -->
      <polygon points="
        0,-410
        355,-205
        355,205
        0,410
        -355,205
        -355,-205
      " fill="none" stroke="#0f172a" stroke-width="40" stroke-linejoin="round" />

      <!-- Inner Dashed Telemetry Ring -->
      <polygon points="
        0,-375
        324,-187
        324,187
        0,375
        -324,187
        -324,-187
      " fill="none" stroke="#22c55e" stroke-width="4.5" opacity="0.6" stroke-dasharray="16, 12" />

      <!-- 100% Watertight Solid Base Silhouette (Zero-Leak) -->
      <path d="
        M 0,-330
        L 70,-310 L 140,-230 L 180,-140 L 250,-50 L 295,70 L 275,200 L 205,295 L 120,270
        L 65,340 L 0,365
        L -65,340 L -120,270 L -205,295 L -275,200 L -295,70 L -250,-50 L -180,-140 L -140,-230 L -70,-310
        Z
      " fill="#0b0f19" />

      <!-- === Low-Poly Faceted Razor Raptor / Falcon === -->

      <!-- Symmetrical Crown / Head Crest -->
      <polygon points="0,-330 35,-260 0,-210 -35,-260" fill="#334155" />
      <polygon points="0,-330 0,-210 35,-260" fill="#475569" />
      <polygon points="0,-330 -35,-260 0,-210" fill="#1e293b" />

      <!-- Lateral Cranial Facets (Right) -->
      <polygon points="0,-330 70,-310 100,-240 35,-260" fill="#475569" />
      <polygon points="35,-260 100,-240 120,-170 50,-180 0,-210" fill="#334155" />

      <!-- Lateral Cranial Facets (Left - Symmetrical) -->
      <polygon points="0,-330 -35,-260 -100,-240 -70,-310" fill="#1e293b" />
      <polygon points="0,-210 -50,-180 -120,-170 -100,-240 -35,-260" fill="#0f172a" />

      <!-- Predatory Almond Optics (Right Eye) -->
      <!-- Sharp angular socket with aggressive predator cant -->
      <polygon points="40,-160 115,-190 145,-135 80,-125 35,-140" fill="#0f172a" />
      <polygon points="52,-158 112,-182 135,-138 85,-130" fill="url(#emerald-glow)" />
      <!-- Slit Pupil & Specular Facet -->
      <polygon points="85,-175 102,-158 92,-138 78,-150" fill="#022c22" />
      <polygon points="68,-168 82,-162 76,-152" fill="#bbf7d0" />

      <!-- Predatory Almond Optics (Left Eye - Symmetrical) -->
      <polygon points="-40,-160 -35,-140 -80,-125 -145,-135 -115,-190" fill="#0b0f19" />
      <polygon points="-52,-158 -85,-130 -135,-138 -112,-182" fill="url(#emerald-optic-left)" />
      <!-- Slit Pupil & Specular Facet -->
      <polygon points="-85,-175 -78,-150 -92,-138 -102,-158" fill="#022c22" />
      <polygon points="-68,-168 -76,-152 -82,-162" fill="#bbf7d0" />

      <!-- Hooked Geometric Raptor Beak (Gold / Amber Razor Accent) -->
      <polygon points="0,-150 40,-120 0,-50 -40,-120" fill="#0f172a" />
      <!-- Top Beak Plates -->
      <polygon points="0,-130 25,-100 0,-50" fill="url(#beak-gold)" />
      <polygon points="0,-130 0,-50 -25,-100" fill="#b45309" />
      <!-- Hooked Beak Tip -->
      <polygon points="0,-50 15,-65 0,-20 -15,-65" fill="#f59e0b" />
      <polygon points="0,-50 0,-20 15,-65" fill="#fbbf24" />
      <polygon points="0,-50 -15,-65 0,-20" fill="#d97706" />

      <!-- Armored Throat & Neck Prow -->
      <polygon points="0,-20 35,40 0,110 -35,40" fill="#1e293b" />
      <polygon points="0,-20 0,110 35,40" fill="#334155" />
      <polygon points="0,-20 -35,40 0,110" fill="#0f172a" />

      <!-- Symmetrical Chest Breastplate (Core Execution Vector) -->
      <polygon points="0,110 50,190 0,270 -50,190" fill="#0f172a" />
      <polygon points="0,110 0,270 50,190" fill="#166534" />
      <polygon points="0,110 -50,190 0,270" fill="#14532d" />

      <polygon points="0,270 35,335 0,365 -35,335" fill="#22c55e" />
      <polygon points="0,270 0,365 35,335" fill="#4ade80" />
      <polygon points="0,270 -35,335 0,365" fill="#16a34a" />

      <!-- Aerodynamic Chevron Shield Wings (Right Flank) -->
      <polygon points="35,40 100,20 120,-30 40,-120 0,-20" fill="#334155" />
      <polygon points="120,-30 180,-140 140,-20 100,20" fill="#475569" />
      <polygon points="180,-140 250,-50 210,30 140,-20" fill="#64748b" />
      <polygon points="250,-50 295,70 240,120 210,30" fill="#22c55e" />
      <polygon points="210,30 240,120 170,180 100,110 35,40" fill="#1e293b" />
      <polygon points="100,110 170,180 130,250 50,190 0,110" fill="#0f172a" />
      <polygon points="240,120 275,200 205,295 130,250 170,180" fill="#16a34a" />
      <polygon points="130,250 205,295 120,270 65,340 35,335 0,270" fill="#15803d" />

      <!-- Aerodynamic Chevron Shield Wings (Left Flank - Symmetrical) -->
      <polygon points="0,-20 -40,-120 -120,-30 -100,20 -35,40" fill="#1e293b" />
      <polygon points="-120,-30 -100,20 -140,-20 -180,-140" fill="#334155" />
      <polygon points="-180,-140 -140,-20 -210,30 -250,-50" fill="#1e293b" />
      <polygon points="-250,-50 -210,30 -240,120 -295,70" fill="#15803d" />
      <polygon points="-35,40 -100,110 -170,180 -240,120 -210,30" fill="#0f172a" />
      <polygon points="0,110 -50,190 -130,250 -170,180 -100,110" fill="#0b0f19" />
      <polygon points="-170,180 -130,250 -205,295 -275,200 -240,120" fill="#14532d" />
      <polygon points="0,270 -35,335 -65,340 -120,270 -205,295 -130,250" fill="#0f172a" />

      <!-- Neon Emerald Slicing Vector Accents on Wing Edges -->
      <polygon points="180,-140 250,-50 220,-40" fill="#86efac" />
      <polygon points="250,-50 295,70 265,60" fill="#4ade80" />
      <polygon points="240,120 275,200 250,180" fill="#86efac" />

      <polygon points="-180,-140 -220,-40 -250,-50" fill="#4ade80" />
      <polygon points="-250,-50 -265,60 -295,70" fill="#22c55e" />
      <polygon points="-240,120 -250,180 -275,200" fill="#4ade80" />

    </g>
  </g>
</svg>`;
}

async function main() {
  const outputDir = path.resolve('B:/workgit/tokenprune/docs/images');
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }

  const svg = buildLogoSvg();
  const svgPath = path.join(outputDir, 'logo.svg');
  const pngPath = path.join(outputDir, 'logo.png');

  fs.writeFileSync(svgPath, svg, 'utf-8');

  const resvg = new Resvg(svg, {
    fitTo: { mode: 'width', value: 1024 },
  });
  const pngData = resvg.render().asPng();
  fs.writeFileSync(pngPath, pngData);

  console.log('✓ Successfully generated Razor Raptor logo.svg and logo.png');
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
