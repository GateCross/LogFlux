import CryptoJS from 'crypto-js';

const KEY = 'LogFlux@AES#2024'; // 16 bytes key - 与后端 config.yaml 中的 AESKey 一致

/**
 * AES-CBC 加密
 * 随机 IV 拼接在密文前面，整体 base64 编码
 */
export function encrypt(str: string) {
  const key = CryptoJS.enc.Utf8.parse(KEY);
  const iv = CryptoJS.lib.WordArray.random(16);
  const srcs = CryptoJS.enc.Utf8.parse(str);
  const encrypted = CryptoJS.AES.encrypt(srcs, key, {
    iv,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7,
  });
  const result = iv.concat(encrypted.ciphertext);
  return CryptoJS.enc.Base64.stringify(result);
}

/**
 * AES-CBC 解密
 */
export function decrypt(str: string) {
  const key = CryptoJS.enc.Utf8.parse(KEY);
  const ciphertext = CryptoJS.enc.Base64.parse(str);
  const iv = CryptoJS.lib.WordArray.create(ciphertext.words.slice(0, 4));
  const encrypted = CryptoJS.lib.WordArray.create(ciphertext.words.slice(4));
  const cipherParams = CryptoJS.CipherParams.create({
    ciphertext: encrypted,
  });
  const decrypted = CryptoJS.AES.decrypt(cipherParams, key, {
    iv,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7,
  });
  return CryptoJS.enc.Utf8.stringify(decrypted).toString();
}
