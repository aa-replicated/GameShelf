(function () {
  'use strict';

  const COLS = 10;
  const ROWS = 12;
  const R = 22;            // bubble radius
  const D = R * 2;         // bubble diameter
  const W = COLS * D;
  const H = (ROWS + 2) * D + 60; // +2 rows buffer + shooter area
  const COLORS = ['#EF4444','#3B82F6','#22C55E','#F59E0B','#A855F7','#EC4899'];
  const SHOOTER_Y = H - 50;
  const SHOOTER_X = W / 2;
  const SPEED = 10;

  let canvas, ctx;
  let grid;           // grid[row][col] = color | null
  let shooter;        // {color, angle}
  let projectile;     // {x, y, dx, dy, color} | null
  let nextColor;
  let score;
  let gameOver;
  let animId;
  let mouseX;

  function init() {
    const root = document.getElementById('game-root');
    root.innerHTML = '';
    root.className = 'flex flex-col items-center gap-2';

    const scoreEl = document.createElement('div');
    scoreEl.id = 'snood-score';
    scoreEl.className = 'text-sm font-medium text-gray-600';
    scoreEl.textContent = 'Score: 0';
    root.appendChild(scoreEl);

    canvas = document.createElement('canvas');
    canvas.width = W;
    canvas.height = H;
    canvas.style.maxWidth = '100%';
    canvas.className = 'rounded-lg border border-gray-200 shadow-sm cursor-crosshair';
    root.appendChild(canvas);
    ctx = canvas.getContext('2d');

    const hint = document.createElement('p');
    hint.className = 'text-xs text-gray-400';
    hint.textContent = 'Click to shoot — match 3+ to pop!';
    root.appendChild(hint);

    grid = initGrid();
    score = 0;
    gameOver = false;
    projectile = null;
    nextColor = randomColor();
    shooter = {color: randomColor(), angle: -Math.PI / 2};
    mouseX = SHOOTER_X;

    canvas.addEventListener('mousemove', e => {
      const rect = canvas.getBoundingClientRect();
      const scaleX = canvas.width / rect.width;
      mouseX = (e.clientX - rect.left) * scaleX;
    });
    canvas.addEventListener('click', shoot);

    animId = requestAnimationFrame(loop);
  }

  function initGrid() {
    const g = [];
    for (let row = 0; row < ROWS; row++) {
      g.push([]);
      for (let col = 0; col < COLS; col++) {
        // Fill top 5 rows with bubbles
        g[row][col] = row < 5 ? randomColor() : null;
      }
    }
    return g;
  }

  function randomColor() {
    return COLORS[Math.floor(Math.random() * COLORS.length)];
  }

  function shoot() {
    if (projectile || gameOver) return;
    const dx = mouseX - SHOOTER_X;
    const dy = -(SHOOTER_Y - 30) ; // aim up
    const angle = Math.atan2(-Math.abs(dy + 20), dx - SHOOTER_X + mouseX - SHOOTER_X);
    // Simplified: shoot toward cursor
    const tx = mouseX, ty = 0;
    const dist = Math.hypot(tx - SHOOTER_X, ty - SHOOTER_Y);
    projectile = {
      x: SHOOTER_X, y: SHOOTER_Y - 30,
      dx: (tx - SHOOTER_X) / dist * SPEED,
      dy: (ty - SHOOTER_Y) / dist * SPEED,
      color: shooter.color
    };
    shooter.color = nextColor;
    nextColor = randomColor();
  }

  function loop() {
    update();
    draw();
    if (!gameOver) animId = requestAnimationFrame(loop);
  }

  function update() {
    if (!projectile) return;

    projectile.x += projectile.dx;
    projectile.y += projectile.dy;

    // Wall bounce
    if (projectile.x - R < 0) { projectile.x = R; projectile.dx = Math.abs(projectile.dx); }
    if (projectile.x + R > W) { projectile.x = W - R; projectile.dx = -Math.abs(projectile.dx); }

    // Ceiling
    if (projectile.y - R <= 0) {
      snap(projectile);
      return;
    }

    // Hit existing bubble
    for (let row = 0; row < ROWS; row++) {
      for (let col = 0; col < COLS; col++) {
        if (!grid[row][col]) continue;
        const {bx, by} = bubblePos(row, col);
        if (Math.hypot(projectile.x - bx, projectile.y - by) < D - 2) {
          snap(projectile);
          return;
        }
      }
    }
  }

  function bubblePos(row, col) {
    const offset = row % 2 === 0 ? 0 : R;
    return {bx: col * D + R + offset, by: row * D + R};
  }

  function snap(p) {
    // Find nearest empty grid cell
    let best = null, bestDist = Infinity;
    for (let row = 0; row < ROWS; row++) {
      for (let col = 0; col < COLS; col++) {
        if (grid[row][col]) continue;
        const {bx, by} = bubblePos(row, col);
        const d = Math.hypot(p.x - bx, p.y - by);
        if (d < bestDist) { bestDist = d; best = {row, col}; }
      }
    }
    if (!best || best.row >= ROWS - 1) {
      endGame(); return;
    }
    grid[best.row][best.col] = p.color;
    projectile = null;

    // Check matches
    const cluster = findCluster(best.row, best.col, p.color);
    if (cluster.length >= 3) {
      cluster.forEach(({row, col}) => { grid[row][col] = null; });
      score += cluster.length * 10 + (cluster.length - 3) * 5;
      removeFloating();
      document.getElementById('snood-score').textContent = `Score: ${score}`;
    }

    // Check if bottom row reached
    for (let col = 0; col < COLS; col++) {
      if (grid[ROWS - 1][col]) { endGame(); return; }
    }

    // Check win
    if (grid.flat().every(c => !c)) {
      score += 200;
      endGame(); return;
    }
  }

  function findCluster(row, col, color) {
    const visited = new Set();
    const stack = [{row, col}];
    const result = [];
    while (stack.length) {
      const {row: r, col: c} = stack.pop();
      const key = `${r},${c}`;
      if (visited.has(key)) continue;
      visited.add(key);
      if (r < 0 || r >= ROWS || c < 0 || c >= COLS) continue;
      if (grid[r][c] !== color) continue;
      result.push({row: r, col: c});
      const neighbors = getNeighbors(r, c);
      neighbors.forEach(n => stack.push(n));
    }
    return result;
  }

  function getNeighbors(row, col) {
    const odd = row % 2 === 1;
    return [
      {row: row - 1, col: col + (odd ? 0 : -1)},
      {row: row - 1, col: col + (odd ? 1 : 0)},
      {row: row,     col: col - 1},
      {row: row,     col: col + 1},
      {row: row + 1, col: col + (odd ? 0 : -1)},
      {row: row + 1, col: col + (odd ? 1 : 0)},
    ].filter(n => n.row >= 0 && n.row < ROWS && n.col >= 0 && n.col < COLS);
  }

  function removeFloating() {
    // BFS from top row — anything not connected is floating
    const connected = new Set();
    const queue = [];
    for (let col = 0; col < COLS; col++) {
      if (grid[0][col]) { queue.push({row: 0, col}); connected.add(`0,${col}`); }
    }
    while (queue.length) {
      const {row, col} = queue.shift();
      getNeighbors(row, col).forEach(({row: nr, col: nc}) => {
        const key = `${nr},${nc}`;
        if (!connected.has(key) && grid[nr] && grid[nr][nc]) {
          connected.add(key);
          queue.push({row: nr, col: nc});
        }
      });
    }
    let floatCount = 0;
    for (let row = 0; row < ROWS; row++) {
      for (let col = 0; col < COLS; col++) {
        if (grid[row][col] && !connected.has(`${row},${col}`)) {
          grid[row][col] = null;
          floatCount++;
        }
      }
    }
    if (floatCount > 0) {
      score += floatCount * 5;
      document.getElementById('snood-score').textContent = `Score: ${score}`;
    }
  }

  function draw() {
    ctx.clearRect(0, 0, W, H);
    ctx.fillStyle = '#1E293B';
    ctx.fillRect(0, 0, W, H);

    // Grid bubbles
    for (let row = 0; row < ROWS; row++) {
      for (let col = 0; col < COLS; col++) {
        if (!grid[row][col]) continue;
        const {bx, by} = bubblePos(row, col);
        drawBubble(bx, by, grid[row][col]);
      }
    }

    // Aim line
    const dx = mouseX - SHOOTER_X;
    const dy = -1;
    const dist = Math.hypot(dx, dy);
    ctx.strokeStyle = 'rgba(255,255,255,0.15)';
    ctx.setLineDash([4, 8]);
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(SHOOTER_X, SHOOTER_Y - 30);
    ctx.lineTo(SHOOTER_X + dx / dist * 120, SHOOTER_Y - 30 + dy / dist * 120);
    ctx.stroke();
    ctx.setLineDash([]);

    // Shooter
    drawBubble(SHOOTER_X, SHOOTER_Y - 30, shooter.color);

    // Next bubble
    ctx.fillStyle = 'rgba(255,255,255,0.3)';
    ctx.font = '10px sans-serif';
    ctx.textAlign = 'left';
    ctx.fillText('NEXT:', 4, SHOOTER_Y + 10);
    drawBubble(50, SHOOTER_Y - 5, nextColor);

    // Projectile
    if (projectile) drawBubble(projectile.x, projectile.y, projectile.color);

    if (gameOver) {
      ctx.fillStyle = 'rgba(0,0,0,0.65)';
      ctx.fillRect(0, 0, W, H);
      ctx.fillStyle = '#FFF';
      ctx.font = 'bold 26px sans-serif';
      ctx.textAlign = 'center';
      ctx.fillText('GAME OVER', W / 2, H / 2 - 10);
      ctx.font = '16px sans-serif';
      ctx.fillText(`Score: ${score}`, W / 2, H / 2 + 20);
    }
  }

  function drawBubble(x, y, color) {
    ctx.beginPath();
    ctx.arc(x, y, R - 1, 0, Math.PI * 2);
    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = 'rgba(255,255,255,0.3)';
    ctx.lineWidth = 1.5;
    ctx.stroke();
    // Shine
    ctx.beginPath();
    ctx.arc(x - R * 0.3, y - R * 0.3, R * 0.25, 0, Math.PI * 2);
    ctx.fillStyle = 'rgba(255,255,255,0.35)';
    ctx.fill();
  }

  function endGame() {
    gameOver = true;
    cancelAnimationFrame(animId);
    canvas.removeEventListener('click', shoot);
    draw();
    setTimeout(() => window.GameShelf.gameOver(score), 700);
  }

  init();
})();
