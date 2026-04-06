(function () {
  'use strict';

  const TILE = 20;
  const COLS = 20;
  const ROWS = 20;
  const W = COLS * TILE;
  const H = ROWS * TILE;
  const TICK_MS = 120;

  let canvas, ctx;
  let snake, dir, nextDir, food, score, gameOver, intervalId;

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
    draw();
  }

  function spawnFood() {
    while (true) {
      const f = {x: Math.floor(Math.random() * COLS), y: Math.floor(Math.random() * ROWS)};
      if (!snake.some(s => s.x === f.x && s.y === f.y)) return f;
    }
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
    } else {
      snake.pop();
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
