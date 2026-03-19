let inMemoryToken: string | null = null;
let loadedFromStorage = false;

const DB_NAME = 'spoutmc-runtime';
const STORE_NAME = 'kv';
const TOKEN_KEY = 'k1';

function hasIndexedDB(): boolean {
  return typeof window !== 'undefined' && typeof window.indexedDB !== 'undefined';
}

function openVault(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = window.indexedDB.open(DB_NAME, 1);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME);
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error ?? new Error('IndexedDB open failed'));
  });
}

async function readTokenFromStorage(): Promise<string | null> {
  if (!hasIndexedDB()) {
    return null;
  }
  const db = await openVault();
  try {
    return await new Promise<string | null>((resolve, reject) => {
      const tx = db.transaction(STORE_NAME, 'readonly');
      const store = tx.objectStore(STORE_NAME);
      const request = store.get(TOKEN_KEY);
      request.onsuccess = () => {
        const value = request.result;
        resolve(typeof value === 'string' && value.trim() !== '' ? value : null);
      };
      request.onerror = () => reject(request.error ?? new Error('IndexedDB read failed'));
    });
  } finally {
    db.close();
  }
}

async function writeTokenToStorage(token: string): Promise<void> {
  if (!hasIndexedDB()) {
    return;
  }
  const db = await openVault();
  try {
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(STORE_NAME, 'readwrite');
      const store = tx.objectStore(STORE_NAME);
      store.put(token, TOKEN_KEY);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error ?? new Error('IndexedDB write failed'));
    });
  } finally {
    db.close();
  }
}

async function clearTokenFromStorage(): Promise<void> {
  if (!hasIndexedDB()) {
    return;
  }
  const db = await openVault();
  try {
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(STORE_NAME, 'readwrite');
      const store = tx.objectStore(STORE_NAME);
      store.delete(TOKEN_KEY);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error ?? new Error('IndexedDB delete failed'));
    });
  } finally {
    db.close();
  }
}

export function peekToken(): string | null {
  return inMemoryToken;
}

export function hasTokenSync(): boolean {
  return inMemoryToken !== null && inMemoryToken !== '';
}

export async function getToken(): Promise<string | null> {
  if (loadedFromStorage) {
    return inMemoryToken;
  }
  try {
    inMemoryToken = await readTokenFromStorage();
  } catch {
    inMemoryToken = null;
  } finally {
    loadedFromStorage = true;
  }
  return inMemoryToken;
}

export async function setToken(token: string): Promise<void> {
  inMemoryToken = token;
  loadedFromStorage = true;
  try {
    await writeTokenToStorage(token);
  } catch {
    // Keep in-memory token even when persistence fails.
  }
}

export async function clearToken(): Promise<void> {
  inMemoryToken = null;
  loadedFromStorage = true;
  try {
    await clearTokenFromStorage();
  } catch {
    // Ignore storage clear failures.
  }
}
