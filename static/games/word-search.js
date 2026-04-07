(function () {
  'use strict';

  const GRID_SIZE = 12;
  const WORD_BANK = [
    'JAVASCRIPT', 'PYTHON', 'GOLANG', 'RUST', 'SWIFT',
    'KOTLIN', 'TYPESCRIPT', 'RUBY', 'SCALA', 'ELIXIR',
    'HASKELL', 'CLOJURE', 'ERLANG', 'OCAML', 'DART',
    'JAVA', 'CSHARP', 'PERL', 'PHP', 'LUA'
  ];
  const NUM_WORDS = 6;
  const DIRECTIONS = [
    [0, 1], [1, 0], [1, 1], [-1, 1],
  ];

  let grid = [];
  let words = [];
  let found = new Set();
  let selecting = false;
  let startCell = null;
  let selectedCells = [];
  let score = 0;
  let startTime = Date.now();
  let gameOver = false;
  const wordCellMap = new Map();

  function pickWords() {
    const shuffled = WORD_BANK.slice().sort(() => Math.random() - 0.5);
    return shuffled.slice(0, NUM_WORDS);
  }

  function makeGrid() {
    return Array.from({length: GRID_SIZE}, () => Array(GRID_SIZE).fill(''));
  }

  function canPlace(g, word, r, c, dr, dc) {
    for (let i = 0; i < word.length; i++) {
      const nr = r + dr * i, nc = c + dc * i;
      if (nr < 0 || nr >= GRID_SIZE || nc < 0 || nc >= GRID_SIZE) return false;
      if (g[nr][nc] !== '' && g[nr][nc] !== word[i]) return false;
    }
    return true;
  }

  function placeWord(g, word) {
    for (let attempt = 0; attempt < 200; attempt++) {
      const [dr, dc] = DIRECTIONS[Math.floor(Math.random() * DIRECTIONS.length)];
      const r = Math.floor(Math.random() * GRID_SIZE);
      const c = Math.floor(Math.random() * GRID_SIZE);
      if (canPlace(g, word, r, c, dr, dc)) {
        for (let i = 0; i < word.length; i++) g[r + dr * i][c + dc * i] = word[i];
        return true;
      }
    }
    return false;
  }

  function fillGrid(g) {
    const LETTERS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
    for (let r = 0; r < GRID_SIZE; r++)
      for (let c = 0; c < GRID_SIZE; c++)
        if (g[r][c] === '') g[r][c] = LETTERS[Math.floor(Math.random() * 26)];
  }

  function initGame() {
    grid = makeGrid();
    words = pickWords();
    const placed = [];
    for (const w of words) {
      if (placeWord(grid, w)) placed.push(w);
    }
    words = placed;
    fillGrid(grid);
    found = new Set();
    wordCellMap.clear();
    score = 0;
    startTime = Date.now();
    gameOver = false;
    render();
    document.addEventListener('mouseup', function() { selecting = false; });
  }

  function cellsInLine(a, b) {
    if (!a || !b) return [];
    const dr = b.r - a.r, dc = b.c - a.c;
    // Only valid lines: horizontal, vertical, or 45-degree diagonal
    if (dr !== 0 && dc !== 0 && Math.abs(dr) !== Math.abs(dc)) return [];
    const stepR = Math.sign(dr), stepC = Math.sign(dc);
    const cells = [];
    let r = a.r, c = a.c;
    while (true) {
      cells.push({r, c});
      if (r === b.r && c === b.c) break;
      r += stepR; c += stepC;
    }
    return cells;
  }

  function cellWord(cells) {
    return cells.map(({r, c}) => grid[r][c]).join('');
  }

  function updateHighlight() {
    const table = document.querySelector('#game-root table');
    if (!table) return;
    table.querySelectorAll('td').forEach(function(cell) {
      const r = +cell.dataset.r, c = +cell.dataset.c;
      const isFound = [...found].some(function(w) {
        const info = wordCellMap.get(w);
        return info && info.some(function(cc) { return cc.r === r && cc.c === c; });
      });
      const isSel = selectedCells.some(function(cc) { return cc.r === r && cc.c === c; });
      cell.className = 'w-8 h-8 text-center font-mono font-bold text-sm cursor-pointer rounded transition-colors';
      if (isFound) cell.classList.add('bg-green-200', 'text-green-800');
      else if (isSel) cell.classList.add('bg-blue-200', 'text-blue-800');
      else cell.classList.add('hover:bg-gray-100');
    });
  }

  function render() {
    const root = document.getElementById('game-root');
    root.innerHTML = '';

    const info = document.createElement('div');
    info.className = 'flex gap-6 mb-4 text-sm font-medium text-gray-600';
    info.innerHTML = '<span>Found: <strong>' + found.size + '/' + words.length + '</strong></span>' +
      '<span>Score: <strong id="ws-score">' + score + '</strong></span>';
    root.appendChild(info);

    const table = document.createElement('table');
    table.className = 'select-none border-collapse mx-auto';
    table.style.userSelect = 'none';

    for (let r = 0; r < GRID_SIZE; r++) {
      const tr = document.createElement('tr');
      for (let c = 0; c < GRID_SIZE; c++) {
        const td = document.createElement('td');
        td.textContent = grid[r][c];
        td.dataset.r = r;
        td.dataset.c = c;
        td.className = 'w-8 h-8 text-center font-mono font-bold text-sm cursor-pointer rounded transition-colors';

        const isFound = [...found].some(function(w) {
          const info = wordCellMap.get(w);
          return info && info.some(function(cc) { return cc.r === r && cc.c === c; });
        });
        if (isFound) {
          td.classList.add('bg-green-200', 'text-green-800');
        } else if (selectedCells.some(function(cc) { return cc.r === r && cc.c === c; })) {
          td.classList.add('bg-blue-200', 'text-blue-800');
        } else {
          td.classList.add('hover:bg-gray-100');
        }
        tr.appendChild(td);
      }
      table.appendChild(tr);
    }
    root.appendChild(table);

    const wordList = document.createElement('div');
    wordList.className = 'mt-4 flex flex-wrap gap-2 justify-center';
    words.forEach(function(w) {
      const span = document.createElement('span');
      span.textContent = w;
      span.className = 'px-3 py-1 rounded-full text-sm font-mono font-bold border ' +
        (found.has(w)
          ? 'bg-green-100 text-green-700 border-green-300 line-through'
          : 'bg-gray-100 text-gray-700 border-gray-200');
      wordList.appendChild(span);
    });
    root.appendChild(wordList);

    attachEvents(table);
  }

  function attachEvents(table) {
    const cells = table.querySelectorAll('td');
    cells.forEach(function(td) {
      td.addEventListener('mousedown', function(e) {
        e.preventDefault();
        selecting = true;
        startCell = {r: +td.dataset.r, c: +td.dataset.c};
        selectedCells = [startCell];
        render();
      });
      td.addEventListener('mouseover', function() {
        if (!selecting) return;
        const cur = {r: +td.dataset.r, c: +td.dataset.c};
        selectedCells = cellsInLine(startCell, cur);
        updateHighlight();
      });
      td.addEventListener('mouseup', function() {
        if (!selecting) return;
        selecting = false;
        checkSelection();
      });
    });
  }

  function checkSelection() {
    const w = cellWord(selectedCells);
    const wr = cellWord(selectedCells.slice().reverse());
    const matched = words.find(function(word) { return word === w || word === wr; });
    if (matched && !found.has(matched)) {
      found.add(matched);
      wordCellMap.set(matched, selectedCells.slice());
      const elapsed = Math.max(1, Math.floor((Date.now() - startTime) / 1000));
      score += Math.max(10, Math.floor(matched.length * 100 / elapsed));
      if (found.size === words.length) endGame();
    }
    selectedCells = [];
    render();
  }

  function endGame() {
    gameOver = true;
    const elapsed = Math.floor((Date.now() - startTime) / 1000);
    score = Math.round(score * (1 + 60 / Math.max(elapsed, 10)));
    render();
    setTimeout(function() { window.GameShelf.gameOver(score); }, 300);
  }

  initGame();
})();
