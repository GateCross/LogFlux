import path from 'node:path';

import { execa } from 'execa';

/**
 * 把指定文件加入 git 暂存区。
 *
 * 原先转引 @changesets/git 的 add，为移除该依赖改为本地实现，
 * 行为保持一致：失败时打印 stderr 并返回 false，不抛异常。
 */
async function add(pathToFile: string, cwd: string): Promise<boolean> {
  const gitCmd = await execa('git', ['add', pathToFile], {
    cwd,
    reject: false,
  });

  if (gitCmd.exitCode !== 0) {
    console.log(pathToFile, gitCmd.stderr.toString());
  }

  return gitCmd.exitCode === 0;
}

/**
 * 获取暂存区文件
 */
async function getStagedFiles(): Promise<string[]> {
  try {
    const { stdout } = await execa('git', [
      '-c',
      'submodule.recurse=false',
      'diff',
      '--staged',
      '--diff-filter=ACMR',
      '--name-only',
      '--ignore-submodules',
      '-z',
    ]);

    let changedList = stdout ? stdout.replace(/\0$/, '').split('\0') : [];
    changedList = changedList.map((item) => path.resolve(process.cwd(), item));
    const changedSet = new Set(changedList);
    changedSet.delete('');
    return [...changedSet];
  } catch (error) {
    console.error('Failed to get staged files:', error);
    return [];
  }
}

export { add, getStagedFiles };
