#!/usr/bin/env node
'use strict';

// 桌面端可执行的 cpkpack。打包规则与 pkg/cpk.Pack 对齐：
// 校验 manifest、剔除包根 server/、生成 manifest.sha256。

const crypto = require('crypto');
const fs = require('fs');
const path = require('path');
const zlib = require('zlib');

const CHECKSUM_FILE = 'manifest.sha256';
/** 头像宽、高分别不得超过此值（像素） */
const MAX_AVATAR_DIM = 256;
/** 常见依赖/缓存目录（点开头目录另由 shouldSkipDir 统一处理） */
const SKIP_DIRS = new Set(['__MACOSX', 'node_modules', 'venv', '__pycache__']);

function main() {
  const args = parseArgs(process.argv.slice(2));
  if (!args.src || !args.out) {
    process.stderr.write('用法: node cpkpack.js -src <包源目录> -out <输出目录>\n');
    process.stderr.write('输出文件名固定为 <id>-<version>.cpk\n');
    process.exit(2);
  }
  try {
    const summary = pack(path.resolve(args.src), path.resolve(args.out));
    process.stdout.write(
      `打包成功: ${summary.outPath}\n` +
        `  id: ${summary.id}\n` +
        `  name: ${summary.name}\n` +
        `  version: ${summary.version}\n` +
        `  skills: ${summary.skills.join(', ') || '(无)'}\n` +
        `  files: ${summary.files}\n`,
    );
  } catch (err) {
    process.stderr.write(`打包失败: ${err.message}\n`);
    process.exit(1);
  }
}

function parseArgs(argv) {
  const out = { src: '', out: '' };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '-src' || a === '--src') out.src = argv[++i] || '';
    else if (a === '-out' || a === '--out') out.out = argv[++i] || '';
    else if (a === '-h' || a === '--help') {
      out.src = '';
      out.out = '';
      return out;
    }
  }
  return out;
}

function pack(srcDir, outDir) {
  let st;
  try {
    st = fs.statSync(srcDir);
  } catch (err) {
    throw new Error(`源目录无效: ${srcDir}`);
  }
  if (!st.isDirectory()) throw new Error(`源目录无效: ${srcDir}`);

  const mfPath = path.join(srcDir, 'manifest.yaml');
  let mfText;
  try {
    mfText = fs.readFileSync(mfPath, 'utf8');
  } catch (err) {
    throw new Error(`源目录缺少 manifest.yaml: ${err.message}`);
  }
  const manifest = parseManifest(mfText);
  validateManifest(manifest);
  const promptPath = path.join(srcDir, manifest.systemPrompt);
  if (!fs.existsSync(promptPath) || !fs.statSync(promptPath).isFile()) {
    throw new Error(`源目录缺少系统提示词文件: ${manifest.systemPrompt}`);
  }
  if (manifest.avatar) {
    const avatarPath = path.join(srcDir, manifest.avatar);
    if (!fs.existsSync(avatarPath) || !fs.statSync(avatarPath).isFile()) {
      throw new Error(`源目录缺少头像文件: ${manifest.avatar}`);
    }
    validateAvatarImage(avatarPath);
  }

  const files = [];
  walk(srcDir, '', files);
  if (files.length === 0) throw new Error('源目录无可打包文件');
  files.sort();

  const checksums = files.map((rel) => `${sha256File(path.join(srcDir, rel))}  ${rel}`);
  const entries = files.map((rel) => ({
    name: rel,
    data: fs.readFileSync(path.join(srcDir, rel)),
  }));
  entries.push({
    name: CHECKSUM_FILE,
    data: Buffer.from(checksums.join('\n') + '\n', 'utf8'),
  });

  const fileName = `${manifest.id}-${manifest.version}.cpk`;
  let outStat;
  try {
    outStat = fs.statSync(outDir);
  } catch (err) {
    outStat = null;
  }
  if (outStat && !outStat.isDirectory()) {
    throw new Error(`-out 须为目录，文件名固定为 ${fileName}: ${outDir}`);
  }
  const outPath = path.join(outDir, fileName);
  fs.mkdirSync(outDir, { recursive: true });
  fs.writeFileSync(outPath, buildZip(entries));

  return {
    id: manifest.id,
    name: manifest.name,
    version: manifest.version,
    skills: skillDirs(files, manifest.skills),
    files: entries.length,
    outPath,
  };
}

function walk(src, rel, out) {
  const dir = rel ? path.join(src, rel) : src;
  for (const ent of fs.readdirSync(dir, { withFileTypes: true })) {
    const name = ent.name;
    const childRel = rel ? `${rel}/${name}` : name;
    if (ent.isSymbolicLink()) {
      let followed;
      try {
        followed = fs.statSync(path.join(src, childRel));
      } catch (err) {
        continue;
      }
      if (followed.isDirectory()) continue;
      if (shouldSkipFile(name)) continue;
      out.push(childRel);
      continue;
    }
    if (ent.isDirectory()) {
      if (shouldSkipDir(name)) continue;
      if (isRootServerDir(childRel)) continue;
      walk(src, childRel, out);
      continue;
    }
    if (!ent.isFile()) continue;
    if (shouldSkipFile(name)) continue;
    out.push(childRel);
  }
}

function validateAvatarImage(filePath) {
  const buf = fs.readFileSync(filePath);
  const dim = readImageSize(buf);
  if (!dim) {
    throw new Error(`头像无法解析为图片: ${path.basename(filePath)}`);
  }
  if (dim.width > MAX_AVATAR_DIM || dim.height > MAX_AVATAR_DIM) {
    throw new Error(
      `头像尺寸不能超过 ${MAX_AVATAR_DIM}×${MAX_AVATAR_DIM}，当前为 ${dim.width}×${dim.height}`,
    );
  }
}

/** @returns {{width:number,height:number}|null} */
function readImageSize(buf) {
  if (!Buffer.isBuffer(buf) || buf.length < 24) return null;
  // PNG
  if (
    buf[0] === 0x89 &&
    buf[1] === 0x50 &&
    buf[2] === 0x4e &&
    buf[3] === 0x47 &&
    buf[4] === 0x0d &&
    buf[5] === 0x0a &&
    buf[6] === 0x1a &&
    buf[7] === 0x0a
  ) {
    return { width: buf.readUInt32BE(16), height: buf.readUInt32BE(20) };
  }
  // GIF
  if (buf[0] === 0x47 && buf[1] === 0x49 && buf[2] === 0x46) {
    return { width: buf.readUInt16LE(6), height: buf.readUInt16LE(8) };
  }
  // JPEG：扫描 SOF0/SOF2
  if (buf[0] === 0xff && buf[1] === 0xd8) {
    let i = 2;
    while (i + 9 < buf.length) {
      if (buf[i] !== 0xff) {
        i++;
        continue;
      }
      const marker = buf[i + 1];
      if (marker === 0xc0 || marker === 0xc2) {
        return { height: buf.readUInt16BE(i + 5), width: buf.readUInt16BE(i + 7) };
      }
      if (marker === 0xd9 || marker === 0xda) break;
      const segLen = buf.readUInt16BE(i + 2);
      if (segLen < 2) break;
      i += 2 + segLen;
    }
  }
  return null;
}

function shouldSkipDir(name) {
  return name.startsWith('.') || SKIP_DIRS.has(name);
}

function shouldSkipFile(name) {
  return name.startsWith('.') || name === CHECKSUM_FILE || name.endsWith('.cpk');
}

function isRootServerDir(rel) {
  return rel === 'server' || rel.startsWith('server/');
}

function skillDirs(files, declared) {
  if (Array.isArray(declared) && declared.length > 0) return declared.slice();
  const seen = new Set();
  const dirs = [];
  for (const rel of files) {
    if (!rel.startsWith('skills/')) continue;
    const rest = rel.slice('skills/'.length);
    const slash = rest.indexOf('/');
    if (slash <= 0) continue;
    const dir = rest.slice(0, slash);
    if (seen.has(dir)) continue;
    if (files.includes(`skills/${dir}/SKILL.md`)) {
      seen.add(dir);
      dirs.push(dir);
    }
  }
  return dirs;
}

function parseManifest(text) {
  const root = {};
  let section = null;
  /** @type {any[] | null} 正在解析的 env[].items */
  let nestedItems = null;
  const lines = String(text).replace(/^\uFEFF/, '').split(/\r?\n/);
  for (const raw of lines) {
    if (!raw.trim() || /^\s*#/.test(raw)) continue;
    const indent = raw.match(/^ */)[0].length;
    const line = raw.trim();
    if (indent === 0) {
      section = null;
      nestedItems = null;
      const m = line.match(/^([A-Za-z0-9_]+)\s*:\s*(.*)$/);
      if (!m) throw new Error(`manifest.yaml 解析失败: ${line}`);
      const key = m[1];
      const val = m[2].trim();
      if (val === '') {
        root[key] = {};
        section = key;
        continue;
      }
      if (val.startsWith('[') && val.endsWith(']')) {
        root[key] = val
          .slice(1, -1)
          .split(',')
          .map(unquote)
          .filter(Boolean);
        continue;
      }
      root[key] = unquote(val);
      continue;
    }
    if (!section) throw new Error(`manifest.yaml 解析失败: ${line}`);

    // env[].items 嵌套列表（缩进更深）
    if (nestedItems && indent >= 6) {
      const nestedObj = line.match(/^- ([A-Za-z0-9_]+)\s*:\s*(.*)$/);
      if (nestedObj) {
        const item = {};
        item[nestedObj[1]] = scalar(nestedObj[2]);
        nestedItems.push(item);
        continue;
      }
      const nm = line.match(/^([A-Za-z0-9_]+)\s*:\s*(.*)$/);
      if (nm && nestedItems.length > 0) {
        nestedItems[nestedItems.length - 1][nm[1]] = scalar(nm[2]);
        continue;
      }
    }

    const objItem = line.match(/^- ([A-Za-z0-9_]+)\s*:\s*(.*)$/);
    if (objItem) {
      nestedItems = null;
      if (!Array.isArray(root[section])) root[section] = [];
      const item = {};
      item[objItem[1]] = scalar(objItem[2]);
      root[section].push(item);
      continue;
    }
    if (line.startsWith('- ')) {
      nestedItems = null;
      if (!Array.isArray(root[section])) root[section] = [];
      root[section].push(unquote(line.slice(2)));
      continue;
    }
    const m = line.match(/^([A-Za-z0-9_]+)\s*:\s*(.*)$/);
    if (!m) throw new Error(`manifest.yaml 解析失败: ${line}`);
    if (Array.isArray(root[section])) {
      const last = root[section][root[section].length - 1];
      if (last && typeof last === 'object' && !Array.isArray(last)) {
        const key = m[1];
        const val = m[2].trim();
        if (key === 'items' && val === '') {
          last.items = [];
          nestedItems = last.items;
          continue;
        }
        nestedItems = null;
        last[key] = scalar(m[2]);
        continue;
      }
    }
    nestedItems = null;
    if (!root[section] || typeof root[section] !== 'object' || Array.isArray(root[section])) {
      root[section] = {};
    }
    root[section][m[1]] = unquote(m[2]);
  }
  return {
    id: str(root.id),
    name: str(root.name),
    description: str(root.description),
    version: str(root.version),
    systemPrompt: str(root.systemPrompt),
    avatar: str(root.avatar),
    minClientVersion: str(root.minClientVersion),
    maxSteps: root.maxSteps === undefined || root.maxSteps === '' ? 0 : Number(root.maxSteps),
    skills: Array.isArray(root.skills) ? root.skills.map(str) : [],
    server: { baseUrl: str(root.server && root.server.baseUrl) },
    env: readEnv(root.env),
  };
}

function validateManifest(m) {
  m.id = m.id.trim();
  m.name = m.name.trim();
  m.description = m.description.trim();
  m.version = m.version.trim();
  m.systemPrompt = m.systemPrompt.trim();
  m.avatar = m.avatar.trim();
  m.minClientVersion = m.minClientVersion.trim();
  m.server.baseUrl = m.server.baseUrl.trim().replace(/\/+$/, '');

  if (!/^[a-z0-9][a-z0-9._-]{0,63}$/.test(m.id)) {
    throw new Error(`manifest: id 无效（需小写字母/数字开头，可含 - _ .，最长 64 位）: "${m.id}"`);
  }
  if (!m.name) throw new Error('manifest: name 不能为空');
  if (Buffer.byteLength(m.description, 'utf8') > 512) {
    throw new Error('manifest: description 不能超过 512 字');
  }
  parseVersion(m.version, 'version');
  if (m.minClientVersion) parseVersion(m.minClientVersion, 'minClientVersion');
  if (!m.systemPrompt) throw new Error('manifest: systemPrompt 不能为空（指向提示词文件，如 prompt.md）');
  assertRel(m.systemPrompt, 'systemPrompt');
  if (m.avatar) assertRel(m.avatar, 'avatar');
  if (!Number.isInteger(m.maxSteps) || m.maxSteps < 0 || m.maxSteps > 999) {
    throw new Error('manifest: maxSteps 需在 0-999 之间');
  }
  validateEnv(m.env);
  if (m.server.baseUrl && !/^https?:\/\//.test(m.server.baseUrl)) {
    throw new Error('manifest: server.baseUrl 需以 http:// 或 https:// 开头');
  }
  for (const s of m.skills) {
    if (!String(s).trim() || /[/\\]/.test(s) || s === '.' || s === '..') {
      throw new Error(`manifest: 技能目录名无效: "${s}"`);
    }
  }
}

function assertRel(p, label) {
  if (p.startsWith('/') || p.startsWith('\\')) {
    throw new Error(`manifest: ${label} 不允许绝对路径: "${p}"`);
  }
  if (p.split(/[/\\]/).includes('..')) {
    throw new Error(`manifest: ${label} 不允许包含 .. 路径穿越: "${p}"`);
  }
}

function parseVersion(s, label) {
  let raw = s.trim().replace(/^v/, '');
  if (!raw) throw new Error(`manifest: ${label} 无效: 版本号为空`);
  const dash = raw.indexOf('-');
  if (dash >= 0) {
    if (!raw.slice(dash + 1)) throw new Error(`manifest: ${label} 无效: 预发布标识为空`);
    raw = raw.slice(0, dash);
  }
  const plus = raw.indexOf('+');
  if (plus >= 0) raw = raw.slice(0, plus);
  const parts = raw.split('.');
  if (parts.length > 3) throw new Error(`manifest: ${label} 无效: 版本号段数过多`);
  for (const p of parts) {
    if (!/^\d+$/.test(p)) throw new Error(`manifest: ${label} 无效: 版本号段无效`);
  }
}

function sha256File(file) {
  const h = crypto.createHash('sha256');
  h.update(fs.readFileSync(file));
  return h.digest('hex');
}

function buildZip(entries) {
  const locals = [];
  const centrals = [];
  let offset = 0;
  const now = dosDateTime(new Date());
  for (const e of entries) {
    const name = Buffer.from(e.name, 'utf8');
    const raw = Buffer.isBuffer(e.data) ? e.data : Buffer.from(e.data);
    const crc = crc32(raw);
    const comp = zlib.deflateRawSync(raw);
    const local = Buffer.concat([
      u32(0x04034b50),
      u16(20),
      u16(0x0800),
      u16(8),
      u16(now.time),
      u16(now.date),
      u32(crc),
      u32(comp.length),
      u32(raw.length),
      u16(name.length),
      u16(0),
      name,
      comp,
    ]);
    const central = Buffer.concat([
      u32(0x02014b50),
      u16(20),
      u16(20),
      u16(0x0800),
      u16(8),
      u16(now.time),
      u16(now.date),
      u32(crc),
      u32(comp.length),
      u32(raw.length),
      u16(name.length),
      u16(0),
      u16(0),
      u16(0),
      u16(0),
      u32(0),
      u32(offset),
      name,
    ]);
    locals.push(local);
    centrals.push(central);
    offset += local.length;
  }
  const cd = Buffer.concat(centrals);
  const eocd = Buffer.concat([
    u32(0x06054b50),
    u16(0),
    u16(0),
    u16(entries.length),
    u16(entries.length),
    u32(cd.length),
    u32(offset),
    u16(0),
  ]);
  return Buffer.concat([...locals, cd, eocd]);
}

function dosDateTime(d) {
  const time = (d.getHours() << 11) | (d.getMinutes() << 5) | Math.floor(d.getSeconds() / 2);
  const date = ((d.getFullYear() - 1980) << 9) | ((d.getMonth() + 1) << 5) | d.getDate();
  return { time, date };
}

function crc32(buf) {
  if (typeof zlib.crc32 === 'function') return zlib.crc32(buf) >>> 0;
  let c = ~0;
  for (const b of buf) {
    c ^= b;
    for (let i = 0; i < 8; i++) c = (c >>> 1) ^ (0xedb88320 & -(c & 1));
  }
  return ~c >>> 0;
}

function u16(n) {
  const b = Buffer.alloc(2);
  b.writeUInt16LE(n & 0xffff);
  return b;
}

function u32(n) {
  const b = Buffer.alloc(4);
  b.writeUInt32LE(n >>> 0);
  return b;
}

function str(v) {
  return v === undefined || v === null ? '' : String(v);
}

function scalar(raw) {
  const v = unquote(raw);
  if (v === 'true') return true;
  if (v === 'false') return false;
  return v;
}

function readEnv(raw) {
  if (raw == null || raw === '') return [];
  if (typeof raw === 'object' && !Array.isArray(raw) && Object.keys(raw).length === 0) return [];
  if (!Array.isArray(raw)) throw new Error('manifest: env 需为环境变量列表');
  return raw.map((item) => {
    if (!item || typeof item !== 'object' || Array.isArray(item) || item.name === undefined) {
      throw new Error('manifest: env 每一项都要包含 name');
    }
    const typeRaw = str(item.type).trim().toLowerCase();
    const type = typeRaw === '' || typeRaw === 'string' ? '' : typeRaw;
    let items = [];
    if (Array.isArray(item.items)) {
      items = item.items.map((col, i) => {
        if (!col || typeof col !== 'object' || Array.isArray(col) || col.name === undefined) {
          throw new Error(`manifest: env ${item.name} 的 items[${i}] 缺少 name`);
        }
        return {
          name: str(col.name),
          label: str(col.label),
          secret: col.secret === true,
          required: col.required === true,
        };
      });
    }
    return {
      name: str(item.name),
      label: str(item.label),
      secret: item.secret === true,
      required: item.required === true,
      type,
      items,
    };
  });
}

function validateEnv(env) {
  const seen = new Set();
  for (const item of env) {
    item.name = item.name.trim();
    item.label = item.label.trim();
    if (!/^[A-Za-z_][A-Za-z0-9_]{0,63}$/.test(item.name)) {
      throw new Error(
        `manifest: env.name 无效（字母或下划线开头，仅含字母、数字、下划线，最长 64 位）: "${item.name}"`,
      );
    }
    if (item.name.toUpperCase().startsWith('CHENGZI_')) {
      throw new Error(`manifest: env.name 不能使用 CHENGZI_ 前缀: "${item.name}"`);
    }
    const key = item.name.toUpperCase();
    if (seen.has(key)) throw new Error(`manifest: env.name 重复: "${item.name}"`);
    seen.add(key);
    if ([...item.label].length > 64) throw new Error('manifest: env.label 不能超过 64 字');

    const typ = item.type === '' || item.type === 'string' ? 'string' : item.type;
    if (typ === 'string') {
      if (item.items && item.items.length > 0) {
        throw new Error(`manifest: env ${item.name} 为 string 时不能声明 items`);
      }
      item.type = '';
      item.items = [];
    } else if (typ === 'list') {
      if (!item.items || item.items.length === 0) {
        throw new Error(`manifest: env ${item.name} 为 list 时必须声明 items`);
      }
      if (item.items.length > 16) {
        throw new Error(`manifest: env ${item.name} 的 items 不能超过 16 列`);
      }
      item.type = 'list';
      item.secret = false;
      const colSeen = new Set();
      for (let i = 0; i < item.items.length; i++) {
        const col = item.items[i];
        col.name = col.name.trim();
        col.label = col.label.trim();
        if (!/^[A-Za-z_][A-Za-z0-9_]{0,63}$/.test(col.name)) {
          throw new Error(`manifest: env ${item.name} 的 items[${i}].name 无效: "${col.name}"`);
        }
        const ck = col.name.toUpperCase();
        if (colSeen.has(ck)) {
          throw new Error(`manifest: env ${item.name} 的 items.name 重复: "${col.name}"`);
        }
        colSeen.add(ck);
        if ([...col.label].length > 64) {
          throw new Error(`manifest: env ${item.name} 的 items.label 不能超过 64 字`);
        }
      }
    } else {
      throw new Error(`manifest: env.type 无效（仅支持 string 或 list）: "${item.type}"`);
    }
  }
}

function unquote(s) {
  s = String(s).trim();
  if ((s.startsWith('"') && s.endsWith('"')) || (s.startsWith("'") && s.endsWith("'"))) {
    return s.slice(1, -1);
  }
  return s;
}

main();
