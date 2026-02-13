export type ExclusionCandidateRemoveType = 'id' | 'tag';

export type ExclusionCandidate = {
  removeType: ExclusionCandidateRemoveType;
  removeValue: string;
};

const exclusionCandidateKeySeparator = '\u0000';

export function buildExclusionCandidateKey(removeType: ExclusionCandidateRemoveType, removeValue: string) {
  return `${removeType}${exclusionCandidateKeySeparator}${removeValue}`;
}

export function parseExclusionCandidateKey(key: string): ExclusionCandidate | null {
  const raw = String(key || '');
  const separatorIndex = raw.indexOf(exclusionCandidateKeySeparator);
  if (separatorIndex <= 0) {
    return null;
  }
  const removeType = raw.slice(0, separatorIndex) as ExclusionCandidateRemoveType;
  const removeValue = raw.slice(separatorIndex + exclusionCandidateKeySeparator.length).trim();
  if ((removeType !== 'id' && removeType !== 'tag') || !removeValue) {
    return null;
  }
  return { removeType, removeValue };
}

export function collectExclusionCandidatesFromFeedbackSuggestion(suggestion: string): ExclusionCandidate[] {
  const raw = String(suggestion || '').trim();
  if (!raw) {
    return [];
  }

  const normalized = raw.replace(/\s+/g, ' ');
  const result: ExclusionCandidate[] = [];
  const seen = new Set<string>();
  const appendCandidate = (removeType: ExclusionCandidateRemoveType, removeValue: string) => {
    const value = String(removeValue || '').trim();
    if (!value) {
      return;
    }
    const key = buildExclusionCandidateKey(removeType, value);
    if (seen.has(key)) {
      return;
    }
    seen.add(key);
    result.push({ removeType, removeValue: value });
  };

  const tagPatterns = [
    /ruleremovebytag\s*[=:：]?\s*['"`]?([a-zA-Z0-9_./:-]+)['"`]?/gi,
    /(?:remove\s*by\s*tag|removebytag|rule\s*tag|tag|标签|移除标签|排除标签|按标签(?:移除|排除)?)\s*[#:：=]?\s*['"`]?([a-zA-Z0-9_./:-]+)['"`]?/gi
  ];
  for (const pattern of tagPatterns) {
    pattern.lastIndex = 0;
    let matched = pattern.exec(normalized);
    for (; matched; matched = pattern.exec(normalized)) {
      appendCandidate('tag', matched[1]);
    }
  }

  const idPatterns = [
    /ruleremovebyid\s*[=:：]?\s*['"`]?(\d{3,7})['"`]?/gi,
    /(?:remove\s*by\s*id|removebyid|rule\s*id|ruleid|规则id|规则编号|按id(?:移除|排除)?|移除规则|排除规则|id)\s*[#:：=]?\s*['"`]?(\d{3,7})['"`]?/gi
  ];
  for (const pattern of idPatterns) {
    pattern.lastIndex = 0;
    let matched = pattern.exec(normalized);
    for (; matched; matched = pattern.exec(normalized)) {
      appendCandidate('id', matched[1]);
    }
  }

  const fallbackRuleIDPattern = /\b(\d{5,7})\b/g;
  let fallbackMatched = fallbackRuleIDPattern.exec(normalized);
  for (; fallbackMatched; fallbackMatched = fallbackRuleIDPattern.exec(normalized)) {
    appendCandidate('id', fallbackMatched[1]);
  }

  return result;
}

export function parseExclusionFromFeedbackSuggestion(suggestion: string): ExclusionCandidate {
  const candidates = collectExclusionCandidatesFromFeedbackSuggestion(suggestion);
  if (candidates.length > 0) {
    return candidates[0];
  }
  return { removeType: 'id', removeValue: '' };
}

export function mergePolicyFeedbackCheckedRowKeys(
  previousCheckedRowKeys: number[],
  currentPageIDs: number[],
  checkedRowKeysInCurrentPage: Array<string | number>
) {
  const selectedKeySet = new Set(previousCheckedRowKeys.filter(id => Number.isInteger(id) && id > 0));
  currentPageIDs.forEach(id => {
    if (Number.isInteger(id) && id > 0) {
      selectedKeySet.delete(id);
    }
  });

  checkedRowKeysInCurrentPage
    .map(item => Number(item))
    .filter(id => Number.isInteger(id) && id > 0)
    .forEach(id => selectedKeySet.add(id));

  return Array.from(selectedKeySet);
}

