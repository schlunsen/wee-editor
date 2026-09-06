/**
 * Unit Tests for a11yHelpers accessibility utilities
 * Testing accessibility compliance checking
 */

import { describe, it, expect, beforeEach } from 'vitest'
import {
  isKeyboardAccessible,
  hasFocusStyling,
  hasAriaLabel,
  getInteractiveElements,
  checkHeadingHierarchy,
  checkImageAltText,
  checkFormLabels,
  getAccessibilityReport,
  auditPageAccessibility,
  checkAriaAttributes,
  validateAriaAttributes
} from '@/utils/a11yHelpers'

describe('a11yHelpers', () => {
  describe('isKeyboardAccessible', () => {
    it('should recognize native interactive elements', () => {
      const button = document.createElement('button')
      expect(isKeyboardAccessible(button)).toBe(true)

      const link = document.createElement('a')
      link.href = '#'
      expect(isKeyboardAccessible(link)).toBe(true)

      const input = document.createElement('input')
      expect(isKeyboardAccessible(input)).toBe(true)
    })

    it('should recognize elements with tabindex >= 0', () => {
      const div = document.createElement('div')
      div.setAttribute('tabindex', '0')
      expect(isKeyboardAccessible(div)).toBe(true)
    })

    it('should recognize elements with interactive roles', () => {
      const div = document.createElement('div')
      div.setAttribute('role', 'button')
      expect(isKeyboardAccessible(div)).toBe(true)
    })

    it('should reject non-interactive elements', () => {
      const div = document.createElement('div')
      expect(isKeyboardAccessible(div)).toBe(false)
    })

    it('should reject elements with negative tabindex', () => {
      const div = document.createElement('div')
      div.setAttribute('tabindex', '-1')
      expect(isKeyboardAccessible(div)).toBe(false)
    })

    it('should handle null elements', () => {
      expect(isKeyboardAccessible(null as any)).toBe(false)
    })
  })

  describe('hasAriaLabel', () => {
    it('should detect aria-label attribute', () => {
      const element = document.createElement('div')
      element.setAttribute('aria-label', 'Close button')
      expect(hasAriaLabel(element)).toBe(true)
    })

    it('should detect aria-labelledby attribute', () => {
      const element = document.createElement('div')
      element.setAttribute('aria-labelledby', 'label-id')
      expect(hasAriaLabel(element)).toBe(true)
    })

    it('should detect text content', () => {
      const element = document.createElement('button')
      element.textContent = 'Click me'
      expect(hasAriaLabel(element)).toBe(true)
    })

    it('should reject elements without label', () => {
      const element = document.createElement('div')
      expect(hasAriaLabel(element)).toBe(false)
    })
  })

  describe('getInteractiveElements', () => {
    it('should find all interactive elements', () => {
      const container = document.createElement('div')

      const button = document.createElement('button')
      const link = document.createElement('a')
      link.href = '#'
      const input = document.createElement('input')

      container.appendChild(button)
      container.appendChild(link)
      container.appendChild(input)

      const interactive = getInteractiveElements(container)
      expect(interactive.length).toBeGreaterThanOrEqual(3)
    })

    it('should find elements with role=button', () => {
      const container = document.createElement('div')
      const button = document.createElement('div')
      button.setAttribute('role', 'button')

      container.appendChild(button)

      const interactive = getInteractiveElements(container)
      expect(interactive).toContain(button)
    })

    it('should return empty array for empty container', () => {
      const container = document.createElement('div')
      const interactive = getInteractiveElements(container)
      expect(interactive).toEqual([])
    })
  })

  describe('checkHeadingHierarchy', () => {
    it('should accept proper heading hierarchy', () => {
      const container = document.createElement('div')

      const h1 = document.createElement('h1')
      const h2 = document.createElement('h2')
      const h3 = document.createElement('h3')

      container.appendChild(h1)
      container.appendChild(h2)
      container.appendChild(h3)

      expect(checkHeadingHierarchy(container)).toBe(true)
    })

    it('should reject skipped heading levels', () => {
      const container = document.createElement('div')

      const h1 = document.createElement('h1')
      const h3 = document.createElement('h3') // Skips h2

      container.appendChild(h1)
      container.appendChild(h3)

      expect(checkHeadingHierarchy(container)).toBe(false)
    })

    it('should accept multiple h2s after h1', () => {
      const container = document.createElement('div')

      const h1 = document.createElement('h1')
      const h2a = document.createElement('h2')
      const h2b = document.createElement('h2')

      container.appendChild(h1)
      container.appendChild(h2a)
      container.appendChild(h2b)

      expect(checkHeadingHierarchy(container)).toBe(true)
    })
  })

  describe('checkImageAltText', () => {
    it('should count images with alt text', () => {
      const container = document.createElement('div')

      const img1 = document.createElement('img')
      img1.alt = 'Alt text'

      const img2 = document.createElement('img')
      img2.alt = 'More alt text'

      container.appendChild(img1)
      container.appendChild(img2)

      const result = checkImageAltText(container)
      expect(result.total).toBe(2)
      expect(result.withAlt).toBe(2)
      expect(result.withoutAlt).toBe(0)
    })

    it('should count images without alt text', () => {
      const container = document.createElement('div')

      const img1 = document.createElement('img')
      const img2 = document.createElement('img')
      img2.alt = 'Has alt'

      container.appendChild(img1)
      container.appendChild(img2)

      const result = checkImageAltText(container)
      expect(result.total).toBe(2)
      expect(result.withAlt).toBe(1)
      expect(result.withoutAlt).toBe(1)
    })
  })

  describe('checkFormLabels', () => {
    it('should count labeled inputs', () => {
      const container = document.createElement('div')

      const input = document.createElement('input')
      input.id = 'input-1'

      const label = document.createElement('label')
      label.setAttribute('for', 'input-1')
      label.textContent = 'Label'

      container.appendChild(label)
      container.appendChild(input)

      const result = checkFormLabels(container)
      expect(result.total).toBe(1)
      expect(result.labeled).toBe(1)
    })

    it('should count unlabeled inputs', () => {
      const container = document.createElement('div')
      const input = document.createElement('input')
      container.appendChild(input)

      const result = checkFormLabels(container)
      expect(result.total).toBe(1)
      expect(result.unlabeled).toBe(1)
    })

    it('should recognize aria-label on inputs', () => {
      const container = document.createElement('div')
      const input = document.createElement('input')
      input.setAttribute('aria-label', 'Email')
      container.appendChild(input)

      const result = checkFormLabels(container)
      expect(result.labeled).toBe(1)
    })
  })

  describe('getAccessibilityReport', () => {
    it('should report keyboard accessibility', () => {
      const button = document.createElement('button')
      button.textContent = 'Click'

      const report = getAccessibilityReport(button)

      expect(report.keyboardAccessible).toBe(true)
      expect(report.hasAriaLabel).toBe(true)
    })

    it('should report missing aria labels', () => {
      const div = document.createElement('div')

      const report = getAccessibilityReport(div)

      expect(report.hasAriaLabel).toBe(false)
      expect(report.issues.some(issue => issue.includes('ARIA'))).toBe(true)
    })

    it('should include all check results', () => {
      const button = document.createElement('button')
      button.textContent = 'Submit'

      const report = getAccessibilityReport(button)

      expect(report).toHaveProperty('keyboardAccessible')
      expect(report).toHaveProperty('hasFocusStyling')
      expect(report).toHaveProperty('hasAriaLabel')
      expect(report).toHaveProperty('contrastLevel')
      expect(report).toHaveProperty('issues')
    })
  })

  describe('auditPageAccessibility', () => {
    it('should perform full page audit', () => {
      const container = document.createElement('div')

      const h1 = document.createElement('h1')
      h1.textContent = 'Title'

      const img = document.createElement('img')
      const input = document.createElement('input')

      container.appendChild(h1)
      container.appendChild(img)
      container.appendChild(input)

      const audit = auditPageAccessibility(container)

      expect(audit).toHaveProperty('headingHierarchy')
      expect(audit).toHaveProperty('imageAltText')
      expect(audit).toHaveProperty('formLabels')
      expect(audit).toHaveProperty('interactiveElements')
      expect(audit).toHaveProperty('issues')
    })

    it('should report missing image alt text', () => {
      const container = document.createElement('div')
      const img = document.createElement('img')
      container.appendChild(img)

      const audit = auditPageAccessibility(container)

      expect(audit.issues.some(issue => issue.includes('images are missing alt text'))).toBe(true)
    })

    it('should report unlabeled form inputs', () => {
      const container = document.createElement('div')
      const input = document.createElement('input')
      container.appendChild(input)

      const audit = auditPageAccessibility(container)

      expect(audit.issues.some(issue => issue.includes('inputs are missing labels'))).toBe(true)
    })
  })

  describe('checkAriaAttributes', () => {
    it('should extract aria attributes', () => {
      const element = document.createElement('div')
      element.setAttribute('aria-label', 'Test')
      element.setAttribute('aria-hidden', 'true')

      const attrs = checkAriaAttributes(element)

      expect(attrs['aria-label']).toBe('Test')
      expect(attrs['aria-hidden']).toBe('true')
    })

    it('should ignore non-aria attributes', () => {
      const element = document.createElement('div')
      element.setAttribute('id', 'test')
      element.setAttribute('aria-label', 'Test')

      const attrs = checkAriaAttributes(element)

      expect(attrs['id']).toBeUndefined()
      expect(attrs['aria-label']).toBe('Test')
    })

    it('should return empty object for no aria attributes', () => {
      const element = document.createElement('div')
      const attrs = checkAriaAttributes(element)

      expect(attrs).toEqual({})
    })
  })

  describe('validateAriaAttributes', () => {
    it('should validate button role', () => {
      const element = document.createElement('div')
      element.setAttribute('role', 'button')

      const validation = validateAriaAttributes(element)

      expect(validation.valid).toBe(false)
      expect(validation.errors.some(err => err.includes('tabindex'))).toBe(true)
    })

    it('should detect conflicting attributes', () => {
      const element = document.createElement('div')
      element.setAttribute('aria-label', 'Test')
      element.setAttribute('aria-labelledby', 'label-id')

      const validation = validateAriaAttributes(element)

      expect(validation.valid).toBe(false)
      expect(validation.errors.some(err => err.includes('both'))).toBe(true)
    })

    it('should pass valid aria attributes', () => {
      const element = document.createElement('button')
      element.setAttribute('aria-label', 'Submit')

      const validation = validateAriaAttributes(element)

      expect(validation.valid).toBe(true)
    })
  })
})
