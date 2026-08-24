import { describe, expect, it } from 'vitest'

import { sortModelsForDisplay } from '@/lib/verified-launch-commands'

describe('sortModelsForDisplay', () => {
  it('lists base models before [1m] tier variants', () => {
    const models = [
      { id: 'glm-5.2', provider: 'Baseten' },
      { id: 'glm-5.2[1m]', provider: 'Baseten' },
      { id: 'kimi-k3[1m]', provider: 'Baseten' },
      { id: 'kimi-k3', provider: 'Baseten' },
      { id: 'gpt-5.5[1m]', provider: 'OpenAI' },
      { id: 'gpt-5.5', provider: 'OpenAI' },
    ]

    expect(sortModelsForDisplay(models).map((model) => model.id)).toEqual([
      'glm-5.2',
      'kimi-k3',
      'gpt-5.5',
      'glm-5.2[1m]',
      'kimi-k3[1m]',
      'gpt-5.5[1m]',
    ])
  })
})
