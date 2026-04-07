(function () {
  'use strict';

  const TILE = 20;
  const COLS = 20;
  const ROWS = 20;
  const W = COLS * TILE;
  const H = ROWS * TILE;
  const TICK_MS = 120;
  const SPECIAL_SPAWN_CHANCE = 0.25; // 25% chance after eating normal food
  const SPECIAL_FOOD_TICKS = 50;     // disappears after 50 ticks

  let canvas, ctx;
  let snake, dir, nextDir, food, score, gameOver, intervalId, specialFood;

  function init() {
    const root = document.getElementById('game-root');
    root.innerHTML = '';
    root.className = 'flex flex-col items-center gap-3';

    // Score display
    const scoreEl = document.createElement('div');
    scoreEl.id = 'snake-score';
    scoreEl.className = 'text-sm font-medium text-gray-600';
    scoreEl.textContent = 'Score: 0';
    root.appendChild(scoreEl);

    // Canvas
    canvas = document.createElement('canvas');
    canvas.width = W;
    canvas.height = H;
    canvas.className = 'rounded-lg border border-gray-200 shadow-sm';
    canvas.style.maxWidth = '100%';
    root.appendChild(canvas);
    ctx = canvas.getContext('2d');

    // Controls hint
    const hint = document.createElement('p');
    hint.className = 'text-xs text-gray-400';
    hint.textContent = '← → ↑ ↓ arrow keys to move';
    root.appendChild(hint);

    resetGame();
    document.removeEventListener('keydown', onKey);
    document.addEventListener('keydown', onKey);
    intervalId = setInterval(tick, TICK_MS);
  }

  function resetGame() {
    snake = [{x: 10, y: 10}, {x: 9, y: 10}, {x: 8, y: 10}];
    dir = {x: 1, y: 0};
    nextDir = {x: 1, y: 0};
    food = spawnFood();
    score = 0;
    gameOver = false;
    specialFood = null;
    draw();
  }

  function spawnFood() {
    const free = [];
    for (let x = 0; x < COLS; x++)
      for (let y = 0; y < ROWS; y++)
        if (!snake.some(function(s) { return s.x === x && s.y === y; })) free.push({x: x, y: y});
    if (free.length === 0) { end(); return {x: 0, y: 0}; }
    return free[Math.floor(Math.random() * free.length)];
  }

  function maybeSpawnSpecial() {
    if (specialFood) return; // only one at a time
    if (Math.random() > SPECIAL_SPAWN_CHANCE) return;
    const type = Math.random() < 0.5 ? 'grow' : 'shrink';
    // Find a free cell not occupied by snake or normal food
    const free = [];
    for (let x = 0; x < COLS; x++)
      for (let y = 0; y < ROWS; y++)
        if (!snake.some(function(s) { return s.x === x && s.y === y; }) &&
            !(food.x === x && food.y === y))
          free.push({x: x, y: y});
    if (free.length === 0) return;
    const pos = free[Math.floor(Math.random() * free.length)];
    specialFood = {x: pos.x, y: pos.y, type: type, ticks: SPECIAL_FOOD_TICKS};
  }

  function onKey(e) {
    const map = {
      ArrowUp:    {x: 0, y: -1},
      ArrowDown:  {x: 0, y:  1},
      ArrowLeft:  {x: -1, y: 0},
      ArrowRight: {x:  1, y: 0},
    };
    const d = map[e.key];
    if (!d) return;
    e.preventDefault();
    // Prevent reversing
    if (d.x === -dir.x && d.y === -dir.y) return;
    nextDir = d;
  }

  function tick() {
    if (gameOver) return;
    dir = nextDir;
    const head = {x: snake[0].x + dir.x, y: snake[0].y + dir.y};

    // Wall collision
    if (head.x < 0 || head.x >= COLS || head.y < 0 || head.y >= ROWS) {
      end(); return;
    }
    // Self collision
    if (snake.some(s => s.x === head.x && s.y === head.y)) {
      end(); return;
    }

    snake.unshift(head);
    if (head.x === food.x && head.y === food.y) {
      score += 10;
      food = spawnFood();
      maybeSpawnSpecial();
    } else if (specialFood && head.x === specialFood.x && head.y === specialFood.y) {
      // Ate special food
      if (specialFood.type === 'grow') {
        score += 30;
        // Add 2 extra segments at the tail (grow by 3 total: head was already added)
        for (let i = 0; i < 2; i++) snake.push({...snake[snake.length - 1]});
      } else {
        // shrink: remove 2 tail segments, min length 3
        score += 5;
        const removeCount = Math.min(2, snake.length - 3);
        for (let i = 0; i < removeCount; i++) snake.pop();
      }
      specialFood = null;
    } else {
      snake.pop();
      // Tick down special food
      if (specialFood) {
        specialFood.ticks--;
        if (specialFood.ticks <= 0) specialFood = null;
      }
    }
    document.getElementById('snake-score').textContent = `Score: ${score}`;
    draw();
  }

  function draw() {
    ctx.fillStyle = '#F9FAFB';
    ctx.fillRect(0, 0, W, H);

    // Grid dots
    ctx.fillStyle = '#E5E7EB';
    for (let x = 0; x < COLS; x++)
      for (let y = 0; y < ROWS; y++)
        ctx.fillRect(x * TILE + 9, y * TILE + 9, 2, 2);

    // Food
    ctx.fillStyle = '#EF4444';
    ctx.beginPath();
    ctx.arc(food.x * TILE + TILE / 2, food.y * TILE + TILE / 2, TILE / 2 - 2, 0, Math.PI * 2);
    ctx.fill();

    // Special food
    if (specialFood) {
      const sx = specialFood.x * TILE + TILE / 2;
      const sy = specialFood.y * TILE + TILE / 2;
      ctx.fillStyle = specialFood.type === 'grow' ? '#FF0000' : '#3B82F6';
      ctx.beginPath();
      ctx.arc(sx, sy, TILE / 2 - 1, 0, Math.PI * 2);
      ctx.fill();
      // Pulsing ring (uses ticks remaining for opacity)
      const pulse = specialFood.ticks / SPECIAL_FOOD_TICKS;
      ctx.strokeStyle = specialFood.type === 'grow'
        ? `rgba(255,100,100,${pulse * 0.8})`
        : `rgba(100,150,255,${pulse * 0.8})`;
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.arc(sx, sy, TILE / 2 + 2, 0, Math.PI * 2);
      ctx.stroke();
    }

    // Snake
    snake.forEach((seg, i) => {
      const progress = 1 - i / snake.length;
      ctx.fillStyle = `hsl(${150 + progress * 30}, 60%, ${40 + progress * 20}%)`;
      ctx.beginPath();
      ctx.roundRect(seg.x * TILE + 1, seg.y * TILE + 1, TILE - 2, TILE - 2, 4);
      ctx.fill();
    });

    if (gameOver) {
      ctx.fillStyle = 'rgba(0,0,0,0.5)';
      ctx.fillRect(0, 0, W, H);
      ctx.fillStyle = '#FFF';
      ctx.font = 'bold 28px sans-serif';
      ctx.textAlign = 'center';
      ctx.fillText('GAME OVER', W / 2, H / 2 - 10);
      ctx.font = '16px sans-serif';
      ctx.fillText(`Score: ${score}`, W / 2, H / 2 + 20);
    }
  }

  function end() {
    gameOver = true;
    clearInterval(intervalId);
    draw();
    document.removeEventListener('keydown', onKey);
    setTimeout(() => window.GameShelf.gameOver(score), 600);
  }

  init();
})();
