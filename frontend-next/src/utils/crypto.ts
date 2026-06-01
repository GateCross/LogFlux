/**
 * 密码加密工具（沿用旧 Vue 版 `frontend/src/utils/crypto.ts` 的 crypto-js 逻辑）。
 *
 * 迁移说明（task 4.1 / Req 3.1）：
 *  - 逐字保留旧实现的 AES-CBC + 随机 IV + Pkcs7 padding 语义，
 *    保证与后端 `common/cryptx/aes.go` 的解密约定一致（IV 前置拼接后 Base64 编码）。
 *  - 密钥 `LogFlux@AES#2024`（16 字节）与后端 `config.yaml` 中的 `AESKey` 保持一致。
 *  - 框架无关纯逻辑，不依赖 React / Umi 运行时，可被单元测试直接覆盖。
 */
import CryptoJS from 'crypto-js';

/** 16 字节 AES 密钥 —— 与后端 config.yaml 中的 AESKey 一致。 */
const KEY = 'LogFlux@AES#2024';

/**
 * 使用 AES-CBC（随机 IV）加密字符串。
 *
 * IV 前置拼接到密文，再整体做 Base64 编码（与后端解密约定一致）。
 *
 * @param str 待加密的明文字符串
 * @returns Base64 编码的 `IV + 密文`
 */
export function encrypt(str: string): string {
  const key = CryptoJS.enc.Utf8.parse(KEY);
  // 生成随机 IV
  const iv = CryptoJS.lib.WordArray.random(16);
  const srcs = CryptoJS.enc.Utf8.parse(str);
  const encrypted = CryptoJS.AES.encrypt(srcs, key, {
    iv,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7,
  });
  // 将 IV 和密文拼接，然后 base64 编码
  const result = iv.concat(encrypted.ciphertext);
  return CryptoJS.enc.Base64.stringify(result);
}

/**
 * 使用 AES-CBC 解密字符串。
 *
 * 输入为 Base64 编码的 `IV + 密文`，先分离前 16 字节 IV 再解密。
 *
 * @param str 待解密的 Base64 字符串
 * @returns 解密后的明文字符串
 */
export function decrypt(str: string): string {
  const key = CryptoJS.enc.Utf8.parse(KEY);
  // 解析 base64，分离 IV 和密文
  const ciphertext = CryptoJS.enc.Base64.parse(str);
  const iv = CryptoJS.lib.WordArray.create(ciphertext.words.slice(0, 4));
  const encrypted = CryptoJS.lib.WordArray.create(ciphertext.words.slice(4));
  const cipherParams = CryptoJS.lib.CipherParams.create({ ciphertext: encrypted });
  const decrypted = CryptoJS.AES.decrypt(cipherParams, key, {
    iv,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7,
  });
  return CryptoJS.enc.Utf8.stringify(decrypted).toString();
}
