const path = require('node:path');
const { TxtDictionary } = require('nlptoolkit-dictionary');
const tdkAllApi = require('tdk-all-api');

const tdkPackageRoot = path.dirname(require.resolve('tdk-all-api/package.json'));
const tdkAxios = require(require.resolve('axios', { paths: [tdkPackageRoot] }));
tdkAxios.defaults.timeout = 5000;
const packageRoot = path.dirname(require.resolve('nlptoolkit-dictionary/package.json'));
const dictionary = new TxtDictionary(
  undefined,
  path.join(packageRoot, 'turkish_dictionary.txt'),
  path.join(packageRoot, 'turkish_misspellings.txt'),
  path.join(packageRoot, 'turkish_morphological_lexicon.txt')
);
const detailCache = new Map();
const DETAIL_TTL = 6 * 60 * 60 * 1000;

function normalizeWord(value) {
  return String(value).normalize('NFC').trim().toLocaleLowerCase('tr-TR');
}

function isTurkishDictionaryWord(value) {
  const word = normalizeWord(value);
  return /^[aâbcçdefgğhıiîjklmnoöprsştuüûvyz]{2,30}$/u.test(word) && Boolean(dictionary.getWord(word));
}

async function lookupTdk(value, provider = tdkAllApi) {
  const word = normalizeWord(value);
  if (!word) return null;
  const useCache = provider === tdkAllApi;
  const cached = useCache ? detailCache.get(word) : null;
  if (cached && Date.now() - cached.time < DETAIL_TTL) return cached.value;
  let timeoutHandle;
  const timeout = new Promise((resolve) => {
    timeoutHandle = setTimeout(() => resolve(null), 12_000);
    timeoutHandle.unref();
  });
  const result = await Promise.race([provider(word).catch(() => null), timeout]);
  clearTimeout(timeoutHandle);
  const valueOrNull = result?.word ? result : null;
  if (useCache) {
    detailCache.set(word, { value: valueOrNull, time: Date.now() });
    if (detailCache.size > 1000) detailCache.delete(detailCache.keys().next().value);
  }
  return valueOrNull;
}

module.exports = { normalizeWord, isTurkishDictionaryWord, lookupTdk };
